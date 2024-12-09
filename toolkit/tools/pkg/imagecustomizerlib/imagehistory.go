// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/file"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safechroot"
)

type HistoryAdditionalFileList []HistoryAdditionalFile
type HistoryDirConfigList []HistoryDirConfig

type HistoryDirConfig struct {
	imagecustomizerapi.DirConfig
	SHA256Hashes map[string]string `yaml:"sha256hash" json:"sha256hash"`
}
type HistoryAdditionalFile struct {
	AdditionalFile imagecustomizerapi.AdditionalFile `yaml:"additionalfile" json:"additionalfile"`
	SHA256Hash     string                            `yaml:"sha256hash" json:"sha256hash"`
}

type ParentImage struct {
	ImageName string `yaml:"imagename" json:"imagename"`
	Hash      string `yaml:"hash" json:"hash"`
}
type ScriptWithHash struct {
	Script     imagecustomizerapi.Script `yaml:"script" json:"script"`
	SHA256Hash string                    `yaml:"sha256hash" json:"sha256hash"`
}
type ScriptsList struct {
	PostCustomization     []ScriptWithHash `yaml:"postCustomization" json:"postCustomization"`
	FinalizeCustomization []ScriptWithHash `yaml:"finalizeCustomization" json:"finalizeCustomization"`
}

type HistoryOS struct {
	ResetBootLoaderType imagecustomizerapi.ResetBootLoaderType `yaml:"resetBootLoaderType" json:"resetBootLoaderType"`
	Hostname            string                                 `yaml:"hostname" json:"hostname"`
	Packages            imagecustomizerapi.Packages            `yaml:"packages" json:"packages"`
	SELinux             imagecustomizerapi.SELinux             `yaml:"selinux" json:"selinux"`
	KernelCommandLine   imagecustomizerapi.KernelCommandLine   `yaml:"kernelCommandLine" json:"kernelCommandLine"`
	AdditionalFiles     HistoryAdditionalFileList              `yaml:"additionalFiles" json:"additionalFiles"`
	AdditionalDirs      HistoryDirConfigList                   `yaml:"additionalDirs" json:"additionalDirs"`
	Users               []imagecustomizerapi.User              `yaml:"users" json:"users"`
	Services            imagecustomizerapi.Services            `yaml:"services" json:"services"`
	Modules             []imagecustomizerapi.Module            `yaml:"modules" json:"modules"`
	Overlays            *[]imagecustomizerapi.Overlay          `yaml:"overlays" json:"overlays"`
}
type HistoryConfig struct {
	Storage imagecustomizerapi.Storage `yaml:"storage" json:"storage"`
	Iso     *imagecustomizerapi.Iso    `yaml:"iso" json:"iso"`
	Pxe     *imagecustomizerapi.Pxe    `yaml:"pxe" json:"pxe"`
	OS      *HistoryOS                 `yaml:"os" json:"os"`
	Scripts ScriptsList                `yaml:"scripts" json:"scripts"`
}
type ImageHistory struct {
	BuildTime     string        `yaml:"datetime" json:"datetime"`
	ToolVersion   string        `yaml:"toolversion" json:"toolversion"`
	ImageUuid     string        `yaml:"imageuuid" json:"imageuuid"`
	ParentImage   ParentImage   `yaml:"parentimage" json:"parentimage"`
	HistoryConfig HistoryConfig `yaml:"config" json:"config"`
}

func populateAdditionalDirs(configAdditionalDirs imagecustomizerapi.DirConfigList, baseConfigPath string) (HistoryDirConfigList, error) {
	var historyDirConfigList = make([]HistoryDirConfig, len(configAdditionalDirs))

	for i := range configAdditionalDirs {
		historyDirConfigList[i].DirConfig = configAdditionalDirs[i]
		hashes := make(map[string]string)
		path := configAdditionalDirs[i].Source
		dirPath := file.GetAbsPathWithBase(baseConfigPath, path)

		// Walk the directory
		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Calculate relative path using baseConfigPath as the root
			relPath, err := filepath.Rel(baseConfigPath, path)
			if err != nil {
				return fmt.Errorf("failed to compute relative path for %s: %w", path, err)
			}

			hash, err := file.GenerateSHA256(path)
			if err != nil {
				return fmt.Errorf("failed to generate SHA256 for file %s: %w", path, err)
			}

			hashes[relPath] = hash
			return nil
		})
		if err != nil {
			return historyDirConfigList, err
		}
		historyDirConfigList[i].SHA256Hashes = hashes
	}
	return historyDirConfigList, nil
}

func populateAdditionalFiles(configAdditionalFiles imagecustomizerapi.AdditionalFileList, baseConfigPath string) (HistoryAdditionalFileList, error) {
	var historyAdditionalFileList = make([]HistoryAdditionalFile, len(configAdditionalFiles))

	for i := range configAdditionalFiles {
		historyAdditionalFileList[i].AdditionalFile = configAdditionalFiles[i]
		if configAdditionalFiles[i].Source == "" {
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, configAdditionalFiles[i].Source)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return historyAdditionalFileList, err
		}
		historyAdditionalFileList[i].SHA256Hash = hash
	}
	return historyAdditionalFileList, nil
}
func populateOS(configOS imagecustomizerapi.OS, baseConfigPath string) (HistoryOS, error) {

	var historyOS HistoryOS
	historyOS.ResetBootLoaderType = configOS.ResetBootLoaderType
	historyOS.Hostname = configOS.Hostname
	historyOS.Packages = configOS.Packages
	historyOS.SELinux = configOS.SELinux
	historyOS.KernelCommandLine = configOS.KernelCommandLine
	historyOS.Users = configOS.Users
	historyOS.Services = configOS.Services
	historyOS.Modules = configOS.Modules
	historyOS.Overlays = configOS.Overlays

	historyAdditionalFileList, err := populateAdditionalFiles(configOS.AdditionalFiles, baseConfigPath)

	if err != nil {
		return historyOS, err
	}

	historyOS.AdditionalFiles = historyAdditionalFileList

	historyAdditionalDirsList, err := populateAdditionalDirs(configOS.AdditionalDirs, baseConfigPath)

	if err != nil {
		return historyOS, err
	}

	historyOS.AdditionalDirs = historyAdditionalDirsList

	return historyOS, nil
}
func populateScriptsList(scripts imagecustomizerapi.Scripts, baseConfigPath string) (ScriptsList, error) {
	var scriptslist ScriptsList
	scriptslist.PostCustomization = make([]ScriptWithHash, len(scripts.PostCustomization))
	scriptslist.FinalizeCustomization = make([]ScriptWithHash, len(scripts.FinalizeCustomization))
	for i := range scripts.PostCustomization {
		scriptslist.PostCustomization[i].Script = scripts.PostCustomization[i]
		path := scripts.PostCustomization[i].Path
		if path == "" {
			// ignore entry if content is provided instead of path
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return scriptslist, err
		}
		scriptslist.PostCustomization[i].SHA256Hash = hash
	}
	for i := range scripts.FinalizeCustomization {
		scriptslist.FinalizeCustomization[i].Script = scripts.FinalizeCustomization[i]
		path := scripts.FinalizeCustomization[i].Path
		if path == "" {
			// ignore entry if content is provided instead of path
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return scriptslist, err
		}
		scriptslist.FinalizeCustomization[i].SHA256Hash = hash

	}

	return scriptslist, nil

}
func addImageHistory(imageChroot *safechroot.Chroot, imageUuid string, inputImageFile string, configFile string, baseConfigPath string, toolVersion string, buildTime string, config *imagecustomizerapi.Config) error {
	var err error
	scriptsList, err := populateScriptsList(config.Scripts, baseConfigPath)
	if err != nil {
		return err
	}
	hash, err := file.GenerateSHA256(inputImageFile)
	if err != nil {
		return err
	}

	historyOS, err := populateOS(*config.OS, baseConfigPath)
	if err != nil {
		return err
	}
	logger.Log.Infof("Creating image customizer history file")
	var allImageHistory []ImageHistory
	var currentImageHistory ImageHistory

	fmt.Println(imageChroot.RootDir())
	customizerLoggingDirPath := filepath.Join(".")
	// customizerLoggingDirPath := filepath.Join(imageChroot.RootDir(), "/etc/image-customizer")
	os.Mkdir(customizerLoggingDirPath, 0755)

	imageHistoryFilePath := filepath.Join(customizerLoggingDirPath, "history.json")

	exists, err := file.PathExists(imageHistoryFilePath)
	if err != nil {
		return err
	}

	if exists {
		file, err := os.ReadFile(imageHistoryFilePath)
		if err != nil {
			log.Fatalf("Error reading file: %v", err)
		}

		// Unmarshal the file content into the data slice
		err = json.Unmarshal(file, &allImageHistory)
		if err != nil {
			log.Fatalf("Error unmarshalling JSON: %v", err)
		}
	}

	currentImageHistory.ImageUuid = imageUuid
	currentImageHistory.ParentImage.Hash = hash
	currentImageHistory.BuildTime = buildTime
	currentImageHistory.ToolVersion = toolVersion
	currentImageHistory.ParentImage.ImageName = filepath.Base(inputImageFile)
	var historyConfig HistoryConfig
	historyConfig.Iso = config.Iso
	historyConfig.OS = &historyOS
	historyConfig.Pxe = config.Pxe
	historyConfig.Scripts = scriptsList

	historyConfig.Storage = config.Storage
	currentImageHistory.HistoryConfig = historyConfig

	allImageHistory = append(allImageHistory, currentImageHistory)

	yamlBytes, err := json.MarshalIndent(allImageHistory, "", " ")
	if err != nil {
		return err
	}
	// jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	// if err != nil {
	// 	return fmt.Errorf("error marshaling JSON: %v", err)
	// }
	file.Write(string(yamlBytes), imageHistoryFilePath)

	return nil
}
