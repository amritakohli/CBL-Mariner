// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.

package imagecustomizerlib

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/file"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/safechroot"
)

type ParentImage struct {
	ImageName string `json:"imagename"`
	Hash      string `json:"hash"`
}
type ScriptFiles struct {
	ScriptType string `json:"type"`
	Path       string `json:"path"`
	Hash       string `json:"hash"`
}
type AdditionalFiles struct {
	SourcePath string `json:"sourcepath"`
	Hash       string `json:"hash"`
}
type ImageHistory struct {
	BuildTime       string            `json:"datetime"`
	ToolVersion     string            `json:"toolversion"`
	ImageUuid       string            `json:"imageuuid"`
	ConfigPath      string            `json:"configpath"`
	ParentImage     ParentImage       `json:"parentimage"`
	ScriptFiles     []ScriptFiles     `json:"scriptfiles"`
	AdditionalFiles []AdditionalFiles `json:"additionalfiles"`
}

func addImageHistory(imageChroot *safechroot.Chroot, imageUuid string, inputImageFile string, configFile string, baseConfigPath string, toolVersion string, buildTime string, additionalFiles imagecustomizerapi.AdditionalFilesMap, scripts *imagecustomizerapi.Scripts) error {
	var err error
	var additionalFilesList []AdditionalFiles
	for sourceFile := range additionalFiles {
		var additionalFile AdditionalFiles
		absSourceFile := file.GetAbsPathWithBase(baseConfigPath, sourceFile)
		hash, err := file.GenerateSHA256(absSourceFile)
		if err != nil {
			return err
		}
		additionalFile.SourcePath = sourceFile
		additionalFile.Hash = hash
		additionalFilesList = append(additionalFilesList, additionalFile)
	}

	var scriptsList []ScriptFiles
	for i := range scripts.PostCustomization {
		var script ScriptFiles
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
		script.Path = path
		script.Hash = hash
		script.ScriptType = "postcustomization"
		scriptsList = append(scriptsList, script)
	}

	for i := range scripts.FinalizeCustomization {
		var script ScriptFiles
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
		script.Path = path
		script.Hash = hash
		script.ScriptType = "finalizecustomization"
		scriptsList = append(scriptsList, script)
	}

	hash, err := file.GenerateSHA256(inputImageFile)
	if err != nil {
		return err
	}

	logger.Log.Infof("Creating image customizer history file")
	var allImageHistory []ImageHistory
	var currentImageHistory ImageHistory
	currentImageHistory.AdditionalFiles = additionalFilesList
	currentImageHistory.ScriptFiles = scriptsList

	fmt.Println(imageChroot.RootDir())
	customizerLoggingDirPath := filepath.Join(imageChroot.RootDir(), "/etc/image-customizer")
	os.Mkdir(customizerLoggingDirPath, 0755)

	imageHistoryFilePath := filepath.Join(customizerLoggingDirPath, "history.json")

	exists, err := file.PathExists(imageHistoryFilePath)
	if err != nil {
		return err
	}

	configNum := 1
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
		configNum = len(allImageHistory) + 1
	}

	configsDirPath := filepath.Join(customizerLoggingDirPath, "configs")
	exists, err = file.DirExists(configsDirPath)
	if err != nil {
		return err
	}
	if !exists {
		// create the directory
		logger.Log.Info("creating configs dir")
		os.Mkdir(configsDirPath, 0755)
	}

	currentImageHistory.ImageUuid = imageUuid
	currentImageHistory.ParentImage.Hash = hash
	currentImageHistory.BuildTime = buildTime
	currentImageHistory.ToolVersion = toolVersion
	currentImageHistory.ParentImage.ImageName = filepath.Base(inputImageFile)
	str := strings.TrimSuffix(filepath.Base(configFile), filepath.Ext(configFile)) + "_config" + strconv.Itoa(configNum) + ".yaml"
	str = filepath.Join(configsDirPath, str)
	configFileToStore, err := os.Create(str)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer configFileToStore.Close()

	currentImageHistory.ConfigPath = configFileToStore.Name()
	file.Copy(configFile, str)

	// Add the new element to the beginning of the array (prepend)
	allImageHistory = append([]ImageHistory{currentImageHistory}, allImageHistory...)
	jsonBytes, err := json.MarshalIndent(allImageHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}
	file.Write(string(jsonBytes), imageHistoryFilePath)

	return nil
}
