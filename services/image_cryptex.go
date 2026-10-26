package services

import (
	"bytes"
	"context"
	"crypto/sha512"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/danielpaulus/go-ios/ios"
)

const maxDDIManifestBytes int64 = 4 << 20
const maxDDIPayloadBytes int64 = 2 << 30

var cryptexPayloads = []struct{ key, filename string }{
	{"Cryptex1,GenericDmg", "Image.dmg"},
	{"Cryptex1,GenericTrustCache", "Image.dmg.trustcache"},
	{"Cryptex1,CryptexInfoPlist", "Image.dmg.cryptex_info"},
	{"Cryptex1,GenericVolume", "Image.dmg.root_hash"},
}

type cryptexAsset struct {
	filename string
	digest   []byte
}

type cryptexRepository struct {
	client  *http.Client
	baseURL string
}

func newCryptexRepository(client *http.Client) cryptexRepository {
	copy := *client
	previousRedirect := copy.CheckRedirect
	copy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("Cryptex 下载重定向必须使用 HTTPS")
		}
		if len(via) >= 10 {
			return fmt.Errorf("Cryptex 下载重定向次数过多")
		}
		if previousRedirect != nil {
			return previousRedirect(req, via)
		}
		return nil
	}
	return cryptexRepository{
		client:  &copy,
		baseURL: "https://raw.githubusercontent.com/doronz88/DeveloperDiskImage/main/PersonalizedImages/Xcode_iOS_DDI_Cryptex/",
	}
}

func parseCryptexAssets(data []byte) ([]cryptexAsset, error) {
	manifest, err := ios.ParsePlist(data)
	if err != nil {
		return nil, fmt.Errorf("解析 Cryptex 清单失败: %w", err)
	}
	if build, ok := manifest["ProductBuildVersion"].(string); !ok || strings.TrimSpace(build) == "" {
		return nil, fmt.Errorf("Cryptex 清单缺少有效构建号")
	}
	identities, _ := manifest["BuildIdentities"].([]interface{})
	for _, raw := range identities {
		identity, _ := raw.(map[string]interface{})
		if identity["Cryptex1,ChipID"] == nil {
			continue
		}
		entries, _ := identity["Manifest"].(map[string]interface{})
		assets := make([]cryptexAsset, 0, len(cryptexPayloads))
		for _, payload := range cryptexPayloads {
			entry, _ := entries[payload.key].(map[string]interface{})
			info, _ := entry["Info"].(map[string]interface{})

			if info["Path"] != payload.filename {
				return nil, fmt.Errorf("Cryptex 清单资源路径无效: %s", payload.key)
			}
			digest, _ := entry["Digest"].([]byte)
			if len(digest) != sha512.Size384 {
				return nil, fmt.Errorf("Cryptex 清单缺少 SHA-384 校验值: %s", payload.key)
			}
			assets = append(assets, cryptexAsset{filename: payload.filename, digest: digest})
		}
		return assets, nil
	}
	return nil, fmt.Errorf("镜像清单没有 Cryptex1 资源")
}

func (r cryptexRepository) ensure(ctx context.Context, baseDir string) (string, error) {
	root := filepath.Join(baseDir, "Xcode_iOS_DDI_Cryptex")
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", fmt.Errorf("创建 Cryptex 缓存目录失败: %w", err)
	}
	info, err := os.Lstat(root)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("Cryptex 缓存路径必须是普通目录: %s", root)
	}
	manifestPath := filepath.Join(root, "BuildManifest.plist")
	if data, err := readDDIManifest(manifestPath); err == nil {
		if assets, err := parseCryptexAssets(data); err == nil {
			valid := true
			for _, asset := range assets {
				ok, err := validCryptexAsset(ctx, filepath.Join(root, asset.filename), asset.digest)
				if err != nil {
					return "", err
				}
				valid = valid && ok
			}
			if valid {
				return root, nil
			}
		}
	} else if os.IsPermission(err) {
		return "", fmt.Errorf("读取 Cryptex 缓存清单失败: %w", err)
	}

	Log.Info("ImageService", "下载或修复 Cryptex 开发者镜像资源")
	response, err := r.get(ctx, "BuildManifest.plist")
	if err != nil {
		return "", err
	}
	data, readErr := io.ReadAll(io.LimitReader(response.Body, maxDDIManifestBytes+1))
	response.Body.Close()
	if readErr != nil {
		return "", fmt.Errorf("读取 Cryptex 清单失败: %w", readErr)
	}
	if int64(len(data)) > maxDDIManifestBytes {
		return "", fmt.Errorf("Cryptex 清单超过大小限制")
	}
	assets, err := parseCryptexAssets(data)
	if err != nil {
		return "", err
	}
	for _, asset := range assets {
		path := filepath.Join(root, asset.filename)
		valid, err := validCryptexAsset(ctx, path, asset.digest)
		if err != nil {
			return "", err
		}
		if valid {
			continue
		}
		response, err := r.get(ctx, asset.filename)
		if err != nil {
			return "", err
		}
		err = writeDDIFile(ctx, path, response.Body, asset.digest, maxDDIPayloadBytes)
		response.Body.Close()
		if err != nil {
			return "", fmt.Errorf("保存 Cryptex 资源 %s 失败: %w", asset.filename, err)
		}
	}
	if err := writeDDIFile(ctx, manifestPath, bytes.NewReader(data), nil, maxDDIManifestBytes); err != nil {
		return "", fmt.Errorf("保存 Cryptex 清单失败: %w", err)
	}
	return root, nil
}

func readDDIManifest(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDDIManifestBytes {
		return nil, fmt.Errorf("Cryptex 缓存清单不是普通文件或超过大小限制")
	}
	return os.ReadFile(path)
}

func (r cryptexRepository) get(ctx context.Context, filename string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(r.baseURL, "/")+"/"+filename, nil)
	if err != nil {
		return nil, fmt.Errorf("创建 Cryptex 下载请求失败: %w", err)
	}
	response, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("下载 Cryptex 资源 %s 失败: %w", filename, err)
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("下载 Cryptex 资源 %s 失败，HTTP %d", filename, response.StatusCode)
	}
	return response, nil
}

func validCryptexAsset(ctx context.Context, path string, expected []byte) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("检查 Cryptex 缓存文件失败: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("Cryptex 缓存资源不是普通文件: %s", path)
	}
	if info.Size() == 0 || info.Size() > maxDDIPayloadBytes {
		return false, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return false, fmt.Errorf("打开 Cryptex 缓存资源失败: %w", err)
	}
	defer file.Close()
	hash := sha512.New384()
	n, err := io.Copy(hash, io.LimitReader(ddiContextReader{ctx: ctx, reader: file}, maxDDIPayloadBytes+1))
	if err != nil {
		return false, fmt.Errorf("校验 Cryptex 缓存资源失败: %w", err)
	}
	return n <= maxDDIPayloadBytes && bytes.Equal(hash.Sum(nil), expected), nil
}

type ddiContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r ddiContextReader) Read(data []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(data)
}

func writeDDIFile(ctx context.Context, path string, input io.Reader, expected []byte, limit int64) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".cryptex-download-*")
	if err != nil {
		return err
	}
	temp := file.Name()
	defer os.Remove(temp)
	defer file.Close()
	hash := sha512.New384()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(ddiContextReader{ctx: ctx, reader: input}, limit+1))
	if err != nil {
		return err
	}
	if n == 0 || n > limit {
		return fmt.Errorf("下载文件为空或超过大小限制")
	}
	if expected != nil && !bytes.Equal(hash.Sum(nil), expected) {
		return fmt.Errorf("SHA-384 校验失败")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
