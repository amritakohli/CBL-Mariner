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

type ParentImage struct {
	ImageName string `yaml:"imagename" json:"imagename"`
	Hash      string `yaml:"hash" json:"hash"`
}

type ImageHistory struct {
	BuildTime   string                    `yaml:"datetime" json:"datetime"`
	ToolVersion string                    `yaml:"toolversion" json:"toolversion"`
	ImageUuid   string                    `yaml:"imageuuid" json:"imageuuid"`
	ParentImage ParentImage               `yaml:"parentimage" json:"parentimage"`
	Config      imagecustomizerapi.Config `yaml:"config" json:"config"`
}

func populateAdditionalDirs(configAdditionalDirs imagecustomizerapi.DirConfigList, baseConfigPath string) error {

	for i := range configAdditionalDirs {
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
			return err
		}
		configAdditionalDirs[i].SHA256HashMap = hashes
	}
	return nil
}

func populateAdditionalFiles(configAdditionalFiles imagecustomizerapi.AdditionalFileList, baseConfigPath string) error {

	for i := range configAdditionalFiles {
		if configAdditionalFiles[i].Source == "" {
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, configAdditionalFiles[i].Source)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return err
		}
		configAdditionalFiles[i].SHA256Hash = hash
	}
	return nil
}
func populateOS(configOS imagecustomizerapi.OS, baseConfigPath string) error {

	populateAdditionalFiles(configOS.AdditionalFiles, baseConfigPath)

	populateAdditionalDirs(configOS.AdditionalDirs, baseConfigPath)
	return nil
}
func populateScriptsList(scripts imagecustomizerapi.Scripts, baseConfigPath string) error {

	for i := range scripts.PostCustomization {
		path := scripts.PostCustomization[i].Path
		if path == "" {
			// ignore entry if content is provided instead of path
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return err
		}
		scripts.PostCustomization[i].SHA256Hash = hash
	}
	for i := range scripts.FinalizeCustomization {
		path := scripts.FinalizeCustomization[i].Path
		if path == "" {
			// ignore entry if content is provided instead of path
			continue
		}
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return err
		}
		scripts.FinalizeCustomization[i].SHA256Hash = hash

	}

	return nil

}
func addImageHistory(imageChroot *safechroot.Chroot, imageUuid string, inputImageFile string, configFile string, baseConfigPath string, toolVersion string, buildTime string, config *imagecustomizerapi.Config) error {
	var err error
	err = populateScriptsList(config.Scripts, baseConfigPath)
	if err != nil {
		return err
	}
	hash, err := file.GenerateSHA256(inputImageFile)
	if err != nil {
		return err
	}

	err = populateOS(*config.OS, baseConfigPath)
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
	currentImageHistory.Config = *config

	allImageHistory = append(allImageHistory, currentImageHistory)

	yamlBytes, err := json.MarshalIndent(allImageHistory, "", " ")
	if err != nil {
		return err
	}
	file.Write(string(yamlBytes), imageHistoryFilePath)

	return nil
}
