package imagecustomizerlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/microsoft/azurelinux/toolkit/tools/imagecustomizerapi"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/file"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/logger"
	"github.com/microsoft/azurelinux/toolkit/tools/internal/shell"
	"github.com/stretchr/testify/assert"
)

func TestCopyBlockDeviceToFile(t *testing.T) {
	var err error
	logger.Log.Infof("test")
	baseImage := checkSkipForCustomizeImage(t, baseImageTypeCoreEfi)

	buildDir := filepath.Join(tmpDir, "TestCustomizeImageKernelCommandLine")
	outImageFilePath := filepath.Join(buildDir, "image.vhd")

	// Customize image.
	config := &imagecustomizerapi.Config{
		OS: imagecustomizerapi.OS{
			KernelCommandLine: imagecustomizerapi.KernelCommandLine{
				ExtraCommandLine: "console=tty0 console=ttyS0",
			},
		},
	}

	err = CustomizeImage(buildDir, buildDir, config, baseImage, nil, outImageFilePath, "raw", "", false, false)
	if !assert.NoError(t, err) {
		return
	}

	imageConnection, err := connectToCoreEfiImage(buildDir, outImageFilePath)
	if !assert.NoError(t, err) {
		return
	}
	defer imageConnection.Close()

	// TODO: create some loop device
	// diskFilePath, err := createFakeEfiImage(buildDir)
	// assert.NoError(t, err)
	// print(diskFilePath)
	// TODO: provide some output dir
	// outDir := tmpDir
	// buildDir := tmpDir

	// TODO: run copy function
	// logger.Log.Infof("test")
	// partitionFilename := "test"
	// fullPath := filepath.Join(outDir, partitionFilename)
	// partitionRawFilepath, err := createTestRawPartitionFile(fullPath)
	// assert.NoError(t, err)

	// print("image connection")
	// imageConnection := NewImageConnection()
	// err = imageConnection.ConnectLoopback(partitionRawFilepath)
	// if err != nil {
	// 	imageConnection.Close()
	// 	fmt.Println("%w", err)
	// 	return
	// }

	// // Create fake disk.
	// diskFilePath, err := createFakeEfiImage(buildDir)
	// if !assert.NoError(t, err) {
	// 	return
	// }
	// logger.Log.Infof("%s", diskFilePath)

	// print("printing")
	// logger.Log.Infof("Devie path is as follows: %s", imageConnection.Loopback().DevicePath())

	// TODO: assert no err
	// TODO: assert full path is as expected
	// TODO: check that full path exists

}

// func createFakeEfiImage(buildDir string) (string, error) {
// 	var err error

// 	err = os.MkdirAll(buildDir, os.ModePerm)
// 	if err != nil {
// 		return "", fmt.Errorf("failed to make build directory (%s):\n%w", buildDir, err)
// 	}

// 	// Use a prototypical Mariner image partition config.
// 	diskConfig := imagecustomizerapi.Disk{
// 		PartitionTableType: imagecustomizerapi.PartitionTableTypeGpt,
// 		MaxSize:            4096,
// 		Partitions: []imagecustomizerapi.Partition{
// 			{
// 				ID:     "boot",
// 				Flags:  []imagecustomizerapi.PartitionFlag{"esp", "boot"},
// 				Start:  1,
// 				End:    ptrutils.PtrTo(uint64(9)),
// 				FsType: "fat32",
// 			},
// 			{
// 				ID:     "rootfs",
// 				Start:  9,
// 				End:    nil,
// 				FsType: "ext4",
// 			},
// 		},
// 	}

// 	partitionSettings := []imagecustomizerapi.PartitionSetting{
// 		{
// 			ID:              "boot",
// 			MountPoint:      "/boot/efi",
// 			MountOptions:    "umask=0077",
// 			MountIdentifier: imagecustomizerapi.MountIdentifierTypeDefault,
// 		},
// 		{
// 			ID:              "rootfs",
// 			MountPoint:      "/",
// 			MountIdentifier: imagecustomizerapi.MountIdentifierTypeDefault,
// 		},
// 	}

// 	rawDisk := filepath.Join(buildDir, "disk.raw")

// 	installOS := func(imageChroot *safechroot.Chroot) error {
// 		// Don't write anything for the OS.
// 		// The createNewImage function will still write the bootloader and fstab file, which will allow the partition
// 		// discovery logic to work. This allows for a limited set of tests to run without needing any of the RPM files.
// 		return nil
// 	}

// 	err = createNewImage(rawDisk, diskConfig, partitionSettings, "efi",
// 		imagecustomizerapi.KernelCommandLine{}, buildDir, testImageRootDirName, installOS)
// 	if err != nil {
// 		return "", err
// 	}

// 	return rawDisk, nil
// }

func TestCompressWithZstd(t *testing.T) {
	// Create test raw partition file
	partitionFilename := "test"
	partitionRawFilepath, err := createTestRawPartitionFile(partitionFilename)
	assert.NoError(t, err)

	// Compress file with zstd and name it with the given output partition file path
	outputPartitionFilepath := "out.raw.zst"
	err = compressWithZstd(partitionRawFilepath, outputPartitionFilepath)
	assert.NoError(t, err)

	// Check that the output file exists as expected
	_, err = os.Stat("out.raw.zst")
	assert.NoError(t, err)
}
func TestAddSkippableFrame(t *testing.T) {
	// Create a skippable frame containing the metadata and prepend the frame to the partition file
	skippableFrameMetadata, err := createSkippableFrameMetadata()
	assert.NoError(t, err)

	// Create test raw partition file
	partitionFilename := "test"
	partitionRawFilepath, err := createTestRawPartitionFile(partitionFilename)
	assert.NoError(t, err)

	// Compress to .raw.zst partition file
	tempPartitionFilepath := testDir + partitionFilename + "_temp.raw.zst"
	err = compressWithZstd(partitionRawFilepath, tempPartitionFilepath)
	assert.NoError(t, err)

	// Test adding the skippable frame
	partitionFilepath, err := addSkippableFrame(tempPartitionFilepath, skippableFrameMetadata, partitionFilename, testDir)
	assert.NoError(t, err)

	// Verify decompression with skippable frame
	err = verifySkippableFrameDecompression(partitionRawFilepath, partitionFilepath)
	assert.NoError(t, err)

	// Verify skippable frame metadata
	err = verifySkippableFrameMetadataFromFile(partitionFilepath, SkippableFrameMagicNumber, SkippableFramePayloadSize, skippableFrameMetadata)
	assert.NoError(t, err)

	// Remove test partition files
	err = os.Remove(partitionRawFilepath)
	assert.NoError(t, err)
	err = os.Remove(tempPartitionFilepath)
	assert.NoError(t, err)
	err = os.Remove(partitionFilepath)
	assert.NoError(t, err)
}

func createTestRawPartitionFile(filename string) (string, error) {
	// Test data
	testData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	// Output file name
	outputFilename := filename + ".raw"

	// Write data to file
	err := os.WriteFile(outputFilename, testData, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to write test data to partition file %s:\n%w", filename, err)
	}
	logger.Log.Infof("Test raw partition file created: %s", outputFilename)
	return outputFilename, nil
}

// Decompress the .raw.zst partition file and verify the hash matches with the source .raw file
func verifySkippableFrameDecompression(rawPartitionFilepath string, rawZstPartitionFilepath string) (err error) {
	// Decompressing .raw.zst file
	decompressedPartitionFilepath := "decompressed.raw"
	err = shell.ExecuteLive(true, "zstd", "-d", rawZstPartitionFilepath, "-o", decompressedPartitionFilepath)
	if err != nil {
		return fmt.Errorf("failed to decompress %s with zstd:\n%w", rawZstPartitionFilepath, err)
	}

	// Calculating hashes
	rawPartitionFileHash, err := file.GenerateSHA256(rawPartitionFilepath)
	if err != nil {
		return fmt.Errorf("error generating SHA256:\n%w", err)
	}
	decompressedPartitionFileHash, err := file.GenerateSHA256(decompressedPartitionFilepath)
	if err != nil {
		return fmt.Errorf("error generating SHA256:\n%w", err)
	}

	// Verifying hashes are equal
	if rawPartitionFileHash != decompressedPartitionFileHash {
		return fmt.Errorf("decompressed partition file hash does not match source partition file hash: %s != %s", decompressedPartitionFileHash, rawPartitionFilepath)
	}
	logger.Log.Debugf("Decompressed partition file hash matches source partition file hash!")

	// Removing decompressed file
	err = os.Remove(decompressedPartitionFilepath)
	if err != nil {
		return fmt.Errorf("failed to remove raw file %s:\n%w", decompressedPartitionFilepath, err)
	}

	return nil
}

// Verifies that the skippable frame has been correctly prepended to the partition file with the correct data
func verifySkippableFrameMetadataFromFile(partitionFilepath string, magicNumber uint32, frameSize uint32, skippableFrameMetadata [SkippableFramePayloadSize]byte) (err error) {
	// Read existing data from the partition file
	existingData, err := os.ReadFile(partitionFilepath)
	if err != nil {
		return fmt.Errorf("failed to read partition file %s:\n%w", partitionFilepath, err)
	}

	// Verify that the skippable frame has been prepended to the partition file by validating magicNumber
	if binary.LittleEndian.Uint32(existingData[0:4]) != magicNumber {
		return fmt.Errorf("skippable frame has not been prepended to the partition file:\n %d != %d", binary.LittleEndian.Uint32(existingData[0:4]), magicNumber)
	}
	logger.Log.Infof("Skippable frame had been correctly prepended to the partition file.")

	// Verify that the skippable frame has the correct frame size by validating frameSize
	if binary.LittleEndian.Uint32(existingData[4:8]) != frameSize {
		return fmt.Errorf("skippable frame frameSize field does not match the defined frameSize:\n %d != %d", binary.LittleEndian.Uint32(existingData[4:8]), frameSize)
	}
	logger.Log.Infof("Skippable frame frameSize field is correct.")

	// Verify that the skippable frame has the correct inserted metadata by validating skippableFrameMetadata
	if !bytes.Equal(existingData[8:8+frameSize], skippableFrameMetadata[:]) {
		return fmt.Errorf("skippable frame metadata does not match the inserted metadata:\n %d != %d", existingData[8:8+frameSize], skippableFrameMetadata[:])
	}
	logger.Log.Infof("Skippable frame is valid and contains the correct metadata!")

	return nil
}
