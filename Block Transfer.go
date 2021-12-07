/*
File Name:  Block Transfer.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"fmt"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/blockchain"
	"github.com/PeernetOfficial/core/protocol"
)

const maxBlockSize = 1 * 1024 * 1024

func blockTransfer(peer *core.PeerInfo, blockNumber uint64) {
	conn, _, err := peer.BlockTransferRequest(peer.PublicKey, 1, maxBlockSize, []protocol.BlockRange{{Offset: uint64(blockNumber), Limit: 1}})
	if err != nil {
		fmt.Printf("Error starting block transfer: %s\n", err.Error())
		return
	}

	data, targetBlock, blockSize, availability, err := protocol.BlockTransferReadBlock(conn, maxBlockSize)
	conn.Close()

	if err != nil {
		fmt.Printf("Error reading block (indicated block size %d) from remote: %s\n", blockSize, err.Error())
		return
	} else if targetBlock.Limit != 1 || targetBlock.Offset != blockNumber {
		fmt.Printf("Error mismatch requested block %d with returned block %d (count %d)\n", blockNumber, targetBlock.Offset, targetBlock.Limit)
		return
	} else if availability == protocol.GetBlockStatusNotAvailable { // Block range not available
		fmt.Printf("Error requested block %d not available\n", blockNumber)
		return
	} else if availability == protocol.GetBlockStatusSizeExceed { // Block range exceeds size limit
		fmt.Printf("Error block %d reported by remote as exceeding size %d (limit %d)\n", blockNumber, blockSize, maxBlockSize)
		return
	} else if availability != protocol.GetBlockStatusAvailable {
		fmt.Printf("Error requested block %d unknown availability indicator %d\n", blockNumber, availability)
		return
	}

	decoded, status, err := blockchain.DecodeBlockRaw(data)

	if err != nil {
		fmt.Printf("Error decoding block: %s\n", err.Error())
		return
	} else if status != blockchain.StatusOK {
		fmt.Printf("Error decoding block status is %d\n", status)
		return
	}

	fmt.Printf("Block %d from %s: version %d, number %d, block size %d, decoded %d records\n", blockNumber, hex.EncodeToString(peer.PublicKey.SerializeCompressed()), decoded.BlockchainVersion, decoded.Number, len(data), len(decoded.RecordsDecoded))

	for _, decodedR := range decoded.RecordsDecoded {
		if file, ok := decodedR.(blockchain.BlockRecordFile); ok {
			blockPrintFile(file)
		} else if recordsProfile, ok := decodedR.([]blockchain.BlockRecordProfile); ok {
			for _, recordP := range recordsProfile {
				blockPrintProfileField(recordP)
			}
		} else {
			fmt.Printf("* Unknown record.\n")
		}
	}
}

func blockPrintFile(file blockchain.BlockRecordFile) {
	fmt.Printf("* File                %s\n", file.ID.String())
	fmt.Printf("  Size                %d\n", file.Size)
	fmt.Printf("  Type                %d\n", file.Type)
	fmt.Printf("  Format              %d\n", file.Format)
	fmt.Printf("  Hash                %s\n", hex.EncodeToString(file.Hash))
	fmt.Printf("  Merkle Root Hash    %s\n", hex.EncodeToString(file.MerkleRootHash))
	fmt.Printf("  Fragment Size       %d\n", file.FragmentSize)

	for _, tag := range file.Tags {
		switch tag.Type {
		case blockchain.TagName:
			fmt.Printf("  Name                %s\n", tag.Text())
		case blockchain.TagFolder:
			fmt.Printf("  Folder              %s\n", tag.Text())
		case blockchain.TagDescription:
			fmt.Printf("  Description         %s\n", tag.Text())
		}
	}
}

func blockPrintProfileField(field blockchain.BlockRecordProfile) {
	switch field.Type {
	case blockchain.ProfileName:
		fmt.Printf("* Profile Name     =  %s\n", string(field.Data))

	case blockchain.ProfileEmail:
		fmt.Printf("* Profile Email    =  %s\n", string(field.Data))

	case blockchain.ProfileWebsite:
		fmt.Printf("* Profile Website  =  %s\n", string(field.Data))

	case blockchain.ProfileTwitter:
		fmt.Printf("* Profile Twitter  =  %s\n", string(field.Data))

	case blockchain.ProfileYouTube:
		fmt.Printf("* Profile YouTube  =  %s\n", string(field.Data))

	case blockchain.ProfileAddress:
		fmt.Printf("* Profile Address  =  %s\n", string(field.Data))

	case blockchain.ProfilePicture:
		fmt.Printf("* Profile Picture. Size %d\n", len(field.Data))

	default:
		fmt.Printf("* Field  %d  =  %s\n", field.Type, hex.EncodeToString(field.Data))
	}
}
