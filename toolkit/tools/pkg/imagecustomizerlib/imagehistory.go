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
	ImageName string `json:"imagename"`
	Hash      string `json:"hash"`
}
type ImageHistory struct {
	BuildTime   string                    `json:"datetime"`
	ToolVersion string                    `json:"toolversion"`
	ImageUuid   string                    `json:"imageuuid"`
	ParentImage ParentImage               `json:"parentimage"`
	ConfigPath  imagecustomizerapi.Config `json:"configpath"`
}

func addImageHistory(imageChroot *safechroot.Chroot, imageUuid string, inputImageFile string, configFile string, baseConfigPath string, toolVersion string, buildTime string, additionalFiles imagecustomizerapi.AdditionalFileList, scripts imagecustomizerapi.Scripts, config *imagecustomizerapi.Config) error {
	var err error
	// var additionalFilesList []AdditionalFiles
	// for i := range additionalFiles {
	// 	var additionalFile AdditionalFiles
	// 	absSourceFile := file.GetAbsPathWithBase(baseConfigPath, additionalFiles[i].Source)
	// 	hash, err := file.GenerateSHA256(absSourceFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	additionalFile.SourcePath = additionalFiles[i].Source
	// 	additionalFile.Hash = hash
	// 	additionalFilesList = append(additionalFilesList, additionalFile)
	// }

	// var scriptsList []ScriptFiles
	// for i := range scripts.PostCustomization {
	// 	var script ScriptFiles
	// 	path := scripts.PostCustomization[i].Path
	// 	if path == "" {
	// 		// ignore entry if content is provided instead of path
	// 		continue
	// 	}
	// 	absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
	// 	hash, err := file.GenerateSHA256(absSourceFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	script.Path = path
	// 	script.Hash = hash
	// 	script.ScriptType = "postcustomization"
	// 	scriptsList = append(scriptsList, script)
	// }

	// for i := range scripts.FinalizeCustomization {
	// 	var script ScriptFiles
	// 	path := scripts.FinalizeCustomization[i].Path
	// 	if path == "" {
	// 		// ignore entry if content is provided instead of path
	// 		continue
	// 	}
	// 	absSourceFile := file.GetAbsPathWithBase(baseConfigPath, path)
	// 	hash, err := file.GenerateSHA256(absSourceFile)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	script.Path = path
	// 	script.Hash = hash
	// 	script.ScriptType = "finalizecustomization"
	// 	scriptsList = append(scriptsList, script)
	// }

	hash, err := file.GenerateSHA256(inputImageFile)
	if err != nil {
		return err
	}

	logger.Log.Infof("Creating image customizer history file")
	var allImageHistory []ImageHistory
	var currentImageHistory ImageHistory
	// currentImageHistory.AdditionalFiles = additionalFilesList
	// currentImageHistory.ScriptFiles = scriptsList

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
	currentImageHistory.ConfigPath = *config

	allImageHistory = append(allImageHistory, currentImageHistory)
	jsonBytes, err := json.MarshalIndent(allImageHistory, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}
	file.Write(string(jsonBytes), imageHistoryFilePath)

	return nil
}
