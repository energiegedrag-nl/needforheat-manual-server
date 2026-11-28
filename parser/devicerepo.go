package parser

import (
	"errors"
	"io/fs"
	"log"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

// DeviceRepoSource is a git repo that contains manuals made by developers of a device.
type DeviceRepoSource struct {
	fs.FS
	deviceName string
	// dir is the directory of the clone; CloseSource removes it.
	dir string
}

// Create a new source filesystem from the firmware repository at url.
//
// deviceName is the device type folder in the manual source (devices/<deviceName>/details.json); the manuals of the
// repository are placed under that name, because that is where the app looks. The repository name is only used when
// deviceName is empty: since the repositories were renamed (twomes-… to needforheat-…) the two no longer match.
func NewDeviceRepoSource(url string, deviceName string, auth transport.AuthMethod) (fs.FS, error) {
	clone, err := newGitFSWithAuth(url, "", auth)
	if err != nil {
		if errors.Is(err, transport.ErrAuthenticationRequired) {
			log.Println("device repo for", url, "could not be opened because it needs authentication")
			return nil, nil
		}

		return nil, err
	}

	if deviceName == "" {
		deviceName = path.Base(url)
	}
	return DeviceRepoSource{FS: clone.fsys, deviceName: deviceName, dir: clone.dir}, nil
}

// Get the path to copy a file to at the destination filesystem.
// Use filePath at sourceFS to determine the path the file should be copied to at the destination filesystem.
func (repo DeviceRepoSource) GetDestinationFilePath(filePath string) string {
	// docs/manuals/installation/assets/
	// docs/manuals/installation/languages/en-US.md
	//
	// devices/Generic-Test/installation/assets/
	// docs/manuals/ need to go
	// and replaced with devices/deviceName

	dir, file := path.Split(filePath)

	dir = strings.TrimSuffix(dir, string(os.PathSeparator))

	splitDirPath := strings.Split(dir, string(os.PathSeparator))

	// Replace docs/manuals with devices/{deviceName}.
	splitDirPath[0] = "devices"
	splitDirPath[1] = repo.deviceName

	// Add "manuafacturer" to the dir path as 'campaign', just before the last element.
	subDir := splitDirPath[len(splitDirPath)-1]
	splitDirPath[len(splitDirPath)-1] = "manufacturer"
	splitDirPath = append(splitDirPath, subDir)

	fullSplitPath := append(splitDirPath, file)

	return path.Join(fullSplitPath...)
}

// Get the path to copy a directory to at the destination filesystem.
// Use dirPath at sourceFS to determine the path the directory should be copied to at the destination filesystem.
func (repo DeviceRepoSource) GetDestinationDirPath(dirPath string) string {
	return dirPath
}
