/*
File Name:  Block Transfer.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/blockchain"
	"github.com/PeernetOfficial/core/protocol"
)

const maxBlockSize = 1 * 1024 * 1024

func blockTransfer(peer *core.PeerInfo, blockNumber uint64, output io.Writer) {
	conn, _, err := peer.BlockTransferRequest(peer.PublicKey, 1, maxBlockSize, []protocol.BlockRange{{Offset: uint64(blockNumber), Limit: 1}})
	if err != nil {
		fmt.Fprintf(output, "Error starting block transfer: %s\n", err.Error())
		return
	}

	data, targetBlock, blockSize, availability, err := protocol.BlockTransferReadBlock(conn, maxBlockSize)
	conn.Close()

	if err != nil {
		fmt.Fprintf(output, "Error reading block (indicated block size %d) from remote: %s\n", blockSize, err.Error())
		return
	} else if targetBlock.Limit != 1 || targetBlock.Offset != blockNumber {
		fmt.Fprintf(output, "Error mismatch requested block %d with returned block %d (count %d)\n", blockNumber, targetBlock.Offset, targetBlock.Limit)
		return
	} else if availability == protocol.GetBlockStatusNotAvailable { // Block range not available
		fmt.Fprintf(output, "Error requested block %d not available\n", blockNumber)
		return
	} else if availability == protocol.GetBlockStatusSizeExceed { // Block range exceeds size limit
		fmt.Fprintf(output, "Error block %d reported by remote as exceeding size %d (limit %d)\n", blockNumber, blockSize, maxBlockSize)
		return
	} else if availability != protocol.GetBlockStatusAvailable {
		fmt.Fprintf(output, "Error requested block %d unknown availability indicator %d\n", blockNumber, availability)
		return
	}

	decoded, status, err := blockchain.DecodeBlockRaw(data)

	if err != nil {
		fmt.Fprintf(output, "Error decoding block: %s\n", err.Error())
		return
	} else if status != blockchain.StatusOK {
		fmt.Fprintf(output, "Error decoding block status is %d\n", status)
		return
	}

	fmt.Fprintf(output, "Block %d from %s: version %d, number %d, block size %d, decoded %d records\n", blockNumber, hex.EncodeToString(peer.PublicKey.SerializeCompressed()), decoded.BlockchainVersion, decoded.Number, len(data), len(decoded.RecordsDecoded))

	for _, decodedR := range decoded.RecordsDecoded {
		if file, ok := decodedR.(blockchain.BlockRecordFile); ok {
			blockPrintFile(file, output)
		} else if recordsProfile, ok := decodedR.([]blockchain.BlockRecordProfile); ok {
			for _, recordP := range recordsProfile {
				blockPrintProfileField(recordP, output)
			}
		} else {
			fmt.Fprintf(output, "* Unknown record.\n")
		}
	}
}

func blockPrintFile(file blockchain.BlockRecordFile, output io.Writer) {
	fmt.Fprintf(output, "* File                %s\n", file.ID.String())
	fmt.Fprintf(output, "  Size                %d\n", file.Size)
	fmt.Fprintf(output, "  Type                %d\n", file.Type)
	fmt.Fprintf(output, "  Format              %d\n", file.Format)
	fmt.Fprintf(output, "  Hash                %s\n", hex.EncodeToString(file.Hash))
	fmt.Fprintf(output, "  Merkle Root Hash    %s\n", hex.EncodeToString(file.MerkleRootHash))
	fmt.Fprintf(output, "  Fragment Size       %d\n", file.FragmentSize)

	for _, tag := range file.Tags {
		switch tag.Type {
		case blockchain.TagName:
			fmt.Fprintf(output, "  Name                %s\n", tag.Text())
		case blockchain.TagFolder:
			fmt.Fprintf(output, "  Folder              %s\n", tag.Text())
		case blockchain.TagDescription:
			fmt.Fprintf(output, "  Description         %s\n", tag.Text())
		}
	}
}

func blockPrintProfileField(field blockchain.BlockRecordProfile, output io.Writer) {
	switch field.Type {
	case blockchain.ProfileName:
		fmt.Fprintf(output, "* Profile Name     =  %s\n", string(field.Data))

	case blockchain.ProfileEmail:
		fmt.Fprintf(output, "* Profile Email    =  %s\n", string(field.Data))

	case blockchain.ProfileWebsite:
		fmt.Fprintf(output, "* Profile Website  =  %s\n", string(field.Data))

	case blockchain.ProfileTwitter:
		fmt.Fprintf(output, "* Profile Twitter  =  %s\n", string(field.Data))

	case blockchain.ProfileYouTube:
		fmt.Fprintf(output, "* Profile YouTube  =  %s\n", string(field.Data))

	case blockchain.ProfileAddress:
		fmt.Fprintf(output, "* Profile Address  =  %s\n", string(field.Data))

	case blockchain.ProfilePicture:
		fmt.Fprintf(output, "* Profile Picture. Size %d\n", len(field.Data))

	default:
		fmt.Fprintf(output, "* Field  %d  =  %s\n", field.Type, hex.EncodeToString(field.Data))
	}
}
