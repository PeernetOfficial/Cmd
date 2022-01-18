/*
File Name:  File Transfer.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"fmt"
	"github.com/PeernetOfficial/core/blockchain"
	"github.com/PeernetOfficial/core/merkle"
	"github.com/google/uuid"
	"io"
	"path/filepath"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/protocol"
	"github.com/PeernetOfficial/core/warehouse"
)

// apiBlockAddFiles contains a list of files from the blockchain
type apiBlockAddFiles struct {
	Files  []apiFile `json:"files"`  // List of files
	Status int       `json:"status"` // Status of the operation, only used when this structure is returned from the API.
}

// apiFile is the metadata of a file published on the blockchain
type apiFile struct {
	ID          uuid.UUID         `json:"id"`          // Unique ID.
	Hash        []byte            `json:"hash"`        // Blake3 hash of the file data
	Type        uint8             `json:"type"`        // File Type. For example audio or document. See TypeX.
	Format      uint16            `json:"format"`      // File Format. This is more granular, for example PDF or Word file. See FormatX.
	Size        uint64            `json:"size"`        // Size of the file
	Folder      string            `json:"folder"`      // Folder, optional
	Name        string            `json:"name"`        // Name of the file
	Description string            `json:"description"` // Description. This is expected to be multiline and contain hashtags!
	Date        time.Time         `json:"date"`        // Date shared
	NodeID      []byte            `json:"nodeid"`      // Node ID, owner of the file. Read only.
	Metadata    []apiFileMetadata `json:"metadata"`    // Additional metadata.
}

// apiFileMetadata contains metadata information.
type apiFileMetadata struct {
	Type uint16 `json:"type"` // See core.TagX constants.
	Name string `json:"name"` // User friendly name of the metadata type. Use the Type fields to identify the metadata as this name may change.
	// Depending on the exact type, one of the below fields is used for proper encoding:
	Text   string    `json:"text"`   // Text value. UTF-8 encoding.
	Blob   []byte    `json:"blob"`   // Binary data
	Date   time.Time `json:"date"`   // Date
	Number uint64    `json:"number"` // Number
}

// Helper function
func blockRecordFileFromAPI(input apiFile) (output blockchain.BlockRecordFile) {
	output = blockchain.BlockRecordFile{ID: input.ID, Hash: input.Hash, Type: input.Type, Format: input.Format, Size: input.Size}

	if input.Name != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagName, input.Name))
	}
	if input.Folder != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagFolder, input.Folder))
	}
	if input.Description != "" {
		output.Tags = append(output.Tags, blockchain.TagFromText(blockchain.TagDescription, input.Description))
	}

	for _, meta := range input.Metadata {
		if blockchain.IsTagVirtual(meta.Type) { // Virtual tags are not mapped back. They are read-only.
			continue
		}

		switch meta.Type {
		case blockchain.TagName, blockchain.TagFolder, blockchain.TagDescription: // auto mapped tags

		case blockchain.TagDateCreated:
			output.Tags = append(output.Tags, blockchain.TagFromDate(meta.Type, meta.Date))

		default:
			output.Tags = append(output.Tags, blockchain.BlockRecordFileTag{Type: meta.Type, Data: meta.Blob})
		}
	}

	return output
}

// IsVirtualFolder returns true if the file is a virtual folder
func (file *apiFile) IsVirtualFolder() bool {
	return file.Type == core.TypeFolder && file.Format == core.FormatFolder
}

// setFileMerkleInfo sets the merkle fields in the BlockRecordFile
func setFileMerkleInfo(backend *core.Backend, file *blockchain.BlockRecordFile) (valid bool) {
	if file.Size <= merkle.MinimumFragmentSize {
		// If smaller or equal than the minimum fragment size, the merkle tree is not used.
		file.MerkleRootHash = file.Hash
		file.FragmentSize = merkle.MinimumFragmentSize
	} else {
		// Get the information from the Warehouse .merkle companion file.
		tree, status, _ := backend.UserWarehouse.ReadMerkleTree(file.Hash, true)
		if status != warehouse.StatusOK {
			return false
		}

		file.MerkleRootHash = tree.RootHash
		file.FragmentSize = tree.FragmentSize
	}

	return true
}


// AddFile Add file to warehouse and blockchain
func AddFile(peer *core.Backend, filePath string, output io.Writer) {
	// Creates a file in the warehouse
	hash, _, err := peer.UserWarehouse.CreateFileFromPath(filePath)
	if err != nil {
		fmt.Fprintf(output, "Error creating file in the warehouse: %s", err)
		return
	}

	// File Successfully added to the warehouse
	fmt.Fprintf(output, "File hash added to warehouse %s \n", hex.EncodeToString(hash))

	// Add the file to the local blockchain
	var input apiBlockAddFiles
	var inputFiles []apiFile
	var inputFile apiFile

	// Write file information to the input file
	inputFile.Date = time.Now()
	// Folder and file name
	dir, file := filepath.Split(filePath)
	inputFile.Folder = dir
	inputFile.Name = file
	inputFile.ID = uuid.New()
	inputFile.Hash = hash

	// Get the public key of the current node
	_, publicKey := peer.ExportPrivateKey()
	inputFile.NodeID = []byte(hex.EncodeToString(publicKey.SerializeCompressed()))

	inputFiles = append(inputFiles, inputFile)

	input.Files = inputFiles

	var filesAdd []blockchain.BlockRecordFile

	for _, file := range input.Files {
		if len(file.Hash) != protocol.HashSize {
			fmt.Fprintf(output, "Bad request")
			return
		}
		if file.ID == uuid.Nil { // if the ID is not provided by the caller, set it
			file.ID = uuid.New()
		}

		// Verify that the file exists in the warehouse. Folders are exempt from this check as they are only virtual.
		if !file.IsVirtualFolder() {
			if _, err := warehouse.ValidateHash(file.Hash); err != nil {
				fmt.Fprintf(output, "Bad request")
				return
			} else if _, fileInfo, status, _ := peer.UserWarehouse.FileExists(file.Hash); status != warehouse.StatusOK {
				//EncodeJSON(api.backend, w, r, apiBlockchainBlockStatus{Status: blockchain.StatusNotInWarehouse})
				fmt.Fprintf(output, "File not in warehouse")
				return
			} else {
				file.Size = uint64(fileInfo.Size())
			}
		} else {
			file.Hash = protocol.HashData(nil)
			file.Size = 0
		}

		blockRecord := blockRecordFileFromAPI(file)

		// Set the merkle tree info as appropriate.
		if !setFileMerkleInfo(peer, &blockRecord) {
			fmt.Fprintf(output, "File not in warehouse")
			return
		}

		filesAdd = append(filesAdd, blockRecord)
	}

	newHeight, newVersion, _ := peer.UserBlockchain.AddFiles(filesAdd)
	fmt.Fprintf(output, "NEW HEIGHT:%d \n", newHeight)
	fmt.Fprintf(output, "NEW VERSION:%d \n", newVersion)
}
