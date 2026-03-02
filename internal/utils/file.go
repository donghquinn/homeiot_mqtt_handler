package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

func ConvertPartFileName(fileName string) string {
	uploadFileName := ConvertNFCToNFD(fileName)
	return ReplaceExtTrim(uploadFileName)
}

func ConvertNFCToNFD(name string) string {
	if norm.NFD.IsNormalString(name) {
		fmt.Printf("Before to converted FileName : %s\n", name)
		t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
		s, _, _ := transform.String(t, name)
		name = s
		fmt.Printf("Complete to converted FileName : %s \n", name)
	}
	return name
}

func ReplaceExtTrim(filename string) string {
	if len(filename) == 0 {
		return ""
	}
	ext := filepath.Ext(filename)
	ext = strings.NewReplacer("%22", "", "\"", "", "%27", "", "'", "", "%20", "", " ", "").Replace(ext)
	ext = strings.TrimSpace(ext)
	return filename[0:len(filename)-len(path.Ext(filename))] + ext
}

/*
v1의 ReadDSirInFile
*/
func ReadFileStat(path string) (os.FileInfo, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read directory %s err: %v", path, err)
	}

	var fileName string

	for _, f := range files {
		fileName = f.Name()
	}

	info, err := os.Stat(fmt.Sprintf("%s/%s", path, fileName))
	if err != nil {
		return nil, fmt.Errorf("check directory %s status err: %v", fmt.Sprintf("%s/%s", path, fileName), err)
	}

	return info, nil
}

/*
v1의 ReadFileStatus
*/
func ReadFileStatus(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		log.Println("Empty FileNames", path)
		return nil, err
	}

	return info, nil
}

// up.UploadReqModel.TempDir
func DeleteTempDir(tmpDir string) error {
	if _, err := os.Stat(tmpDir); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("delete temp dir %s not found", tmpDir)
		}

		return fmt.Errorf("read %s stats got error: %v", tmpDir, err)
	}

	if err := os.RemoveAll(tmpDir + "/"); err != nil {
		return fmt.Errorf("remove %s err: %v", tmpDir, err)
	}

	return nil
}

// func DetectFileType(fileName string) string {
// 	ext := strings.Split(filepath.Ext(fileName), ".")[1]

// 	for _, fileType := range constant.ImageTypeList {
// 		if strings.Contains(ext, fileType) {
// 			return "image"
// 		}
// 	}

// 	for _, fileType := range constant.VideoTypeList {
// 		if strings.Contains(ext, fileType) {
// 			return "video"
// 		}
// 	}

// 	return ""
// }

// ChownPermission changes ownership of path to specified UID/GID with fallback
func ChownPermission(path string, uid, gid int) error {
	// Check if we can actually change ownership
	if !canChangeOwnership(uid, gid) {
		log.Printf("[WARNING] Cannot change ownership to %d:%d, skipping chown for %s", uid, gid, path)
		return nil
	}
	return chownRecursive(path, uid, gid)
}

// chownRecursive recursively changes ownership of directory and all its contents
func chownRecursive(path string, uid, gid int) error {
	// Change ownership of the root path
	if err := os.Chown(path, uid, gid); err != nil {
		return fmt.Errorf("chown %s err: %v", path, err)
	}

	// Walk through directory tree
	return filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if err := os.Chown(filePath, uid, gid); err != nil {
			return fmt.Errorf("chown %s err: %v", filePath, err)
		}

		return nil
	})
}

// CreateDirectoryWithPermissions creates a directory with proper permissions and ownership
func CreateDirectoryWithPermissions(path string, uid, gid int) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create directory %s err: %v", path, err)
	}

	// Try to set ownership, but don't fail if we can't
	if err := ChownPermission(path, uid, gid); err != nil {
		log.Printf("[WARNING] Could not set ownership on %s: %v", path, err)
		// Don't return error - directory was created successfully
	}

	return nil
}

// CreateFileWithPermissions creates a file with proper permissions and ownership
func CreateFileWithPermissions(path string, uid, gid int, perm os.FileMode) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return nil, fmt.Errorf("create file %s err: %v", path, err)
	}

	// Try to set ownership, but don't fail if we can't
	if canChangeOwnership(uid, gid) {
		if err := file.Chown(uid, gid); err != nil {
			log.Printf("[WARNING] Could not set ownership on %s: %v", path, err)
			// Don't close file or return error - file was created successfully
		}
	} else {
		log.Printf("[WARNING] Cannot change ownership to %d:%d, skipping chown for %s", uid, gid, path)
	}

	return file, nil
}

// canChangeOwnership checks if the current user can change ownership to the specified UID/GID
func canChangeOwnership(targetUID, targetGID int) bool {
	// Get current user
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("[WARNING] Could not get current user: %v", err)
		return false
	}

	// Convert current UID to int
	currentUID, err := strconv.Atoi(currentUser.Uid)
	if err != nil {
		log.Printf("[WARNING] Could not parse current UID: %v", err)
		return false
	}

	// Root can change ownership to anyone
	if currentUID == 0 {
		return true
	}

	// Non-root users can only change ownership to themselves
	// Check if we're trying to change to the same user
	if currentUID == targetUID {
		// For GID, check if user belongs to the target group
		groupIDs, err := currentUser.GroupIds()
		if err != nil {
			log.Printf("[WARNING] Could not get user groups: %v", err)
			return false
		}

		targetGIDStr := strconv.Itoa(targetGID)
		for _, gid := range groupIDs {
			if gid == targetGIDStr {
				return true
			}
		}
	}

	// Cannot change ownership
	return false
}

func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if _, err := sourceFile.Stat(); err != nil {
		return fmt.Errorf("read source file stat err: %w", err)
	}

	// 대상 디렉토리 생성
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create target file err: %v", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("copy file err: %w", err)
	}

	// 6. **중요**: 원본 파일의 권한을 대상 파일에 적용
	if err := destFile.Chmod(0644); err != nil {
		return fmt.Errorf("set target file permission 0644 err: %w", err)
	}

	return nil
}
