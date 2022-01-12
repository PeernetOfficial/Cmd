/*
File Name:  File Transfer.go
Copyright:  2021 Peernet Foundation s.r.o.
Author:     Peter Kleissner
*/

package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/PeernetOfficial/core"
	"github.com/PeernetOfficial/core/protocol"
	"github.com/PeernetOfficial/core/udt"
	"github.com/PeernetOfficial/core/warehouse"
)

// transferCompareFile downloads a file from a remote peer and compares it with the same file in the local warehouse.
// This function exists to test a file transfer.
// Note: The file MUST be stored locally, otherwise this function fails.
func transferCompareFile(peer *core.PeerInfo, fileHash []byte, output io.Writer) {
	// check if the file exists locally
	_, fileInfo, status, _ := peer.Backend.UserWarehouse.FileExists(fileHash)
	if status != warehouse.StatusOK {
		fmt.Fprintf(output, "File does not exist in local warehouse: %s\n", hex.EncodeToString(fileHash))
		return
	}
	expectedSize := fileInfo.Size()

	// peer must be connected
	if !peer.IsConnectionActive() {
		fmt.Fprintf(output, "Peer has no active connection: %s\n", hex.EncodeToString(peer.NodeID))
		return
	}

	fmt.Fprintf(output, "1. Peer connected: %s\n", hex.EncodeToString(peer.NodeID))

	// request file transfer
	udtConn, virtualConn, err := peer.FileTransferRequestUDT(fileHash, 0, 0)
	if err != nil {
		fmt.Fprintf(output, "Error opening UDT connection: %s\n", err)
		return
	}
	defer udtConn.Close()

	fmt.Fprintf(output, "2. Opened UDT connection for file: %s\n", hex.EncodeToString(fileHash))

	fileSize, transferSize, err := protocol.FileTransferReadHeader(udtConn)
	if err != nil {
		fmt.Fprintf(output, "Error reading file transfer header: %s\n", err)
		return
	}

	if fileSize != uint64(expectedSize) {
		fmt.Fprintf(output, "Error expected local file size %d mismatch with remote file size %d\n", expectedSize, fileSize)
		return
	} else if fileSize != transferSize {
		fmt.Fprintf(output, "Error remote peer only offering %d of total file size %d\n", transferSize, fileSize)
		return
	}

	fmt.Fprintf(output, "3. Matching transfer size %d and file size %d\n", transferSize, expectedSize)

	// Previous: Loop in explicitly 512 bytes (which is the same buffer as io.Copy apparently) and compare with what is expected.
	// Now use 4 KB buffer.
	fileOffset := 0
	totalRead := 0
	timeStart := time.Now()
	timeUpdateLast := time.Now()
	dataRemaining := fileSize

	for {
		maxSize := uint64(4096)
		if dataRemaining < maxSize {
			maxSize = dataRemaining
		}

		data := make([]byte, maxSize)
		n, err := udtConn.Read(data)

		totalRead += n
		dataRemaining -= uint64(n)
		data = data[:n]

		if err != nil {
			fmt.Fprintf(output, "-- TERMINATE: ERROR READING. Read %d bytes. Total read %d : %v\n", n, fileOffset+n, err)
			break
		} else if n == 0 {
			fmt.Fprintf(output, "-- TERMINATE: EMPTY READ but no error indicated. Read %d bytes. Total read %d : %v\n", n, fileOffset+n, err)
			break
		} else if dataRemaining <= 0 {
			fmt.Fprintf(output, "-- TERMINATE: EVERYTHING READ. Read %d bytes. Total read %d : %v\n", n, fileOffset+n, err)
			break
		}

		// read the exact piece from the local file for comparison
		dataCompare := make([]byte, 0, n)
		compareBuffer := bytes.NewBuffer(dataCompare)
		_, bytesRead, err := peer.Backend.UserWarehouse.ReadFile(fileHash, int64(fileOffset), int64(n), compareBuffer)
		if err != nil {
			fmt.Fprintf(output, "Warehouse error reading at offset %d length %d: %v\n", fileOffset, n, err)
			break
		} else if int(bytesRead) != n {
			fmt.Fprintf(output, "Warehouse did not read full data. Requested %d, provided %d.\n", n, bytesRead)
			break
		}
		dataCompare = compareBuffer.Bytes()

		// make the comparison
		if !bytes.Equal(data, dataCompare) {
			fmt.Fprintf(output, "Offset %08X   read %d   DATA MISMATCH:\n", fileOffset, n)
			fmt.Fprintf(output, "---- DATA FROM REMOTE:\n%s\n", hex.Dump(data))
			fmt.Fprintf(output, "---- DATA FROM LOCAL WAREHOUSE:\n%s\n", hex.Dump(dataCompare))

			break
		}

		// status update every few seconds
		//fmt.Fprintf(output, "Offset %08X   read %d   SUCCESS\n", fileOffset, n)
		if time.Now().After(timeUpdateLast.Add(time.Second)) {
			speed := float64(totalRead) / time.Since(timeStart).Seconds() / 1024
			fmt.Fprintf(output, "Offset %08X   progress %.2f %%   MATCHING. Speed: %.2f KB/s\n", fileOffset, float64((fileOffset+n)*100)/float64(fileSize), speed)
			timeUpdateLast = time.Now()
		}

		fileOffset += n
	}

	fmt.Fprintf(output, "Terminate reason %d: %s\n", virtualConn.GetTerminateReason(), translateTerminateReason(virtualConn.GetTerminateReason()))

	speed := float64(totalRead) / time.Since(timeStart).Seconds() / 1024

	fmt.Fprintf(output, "Transfer took %s. Speed is %.2f KB/s\n", time.Since(timeStart).String(), speed)

	if totalRead != int(expectedSize) {
		fmt.Fprintf(output, "Error transferred data %d mismatch with reported file size %d\n", totalRead, fileSize)
		return
	}

	fmt.Fprintf(output, "Finished reading total of %d bytes. Expected %d bytes.\n", totalRead, fileSize)
}

func translateTerminateReason(reason int) string {
	switch reason {
	case 0:
		return "Virtual connection does not indicated a shutdown."
	case 404:
		return "Remote peer does not store the file."
	case 2:
		return "Remote termination signal (upstream)"
	case 3:
		return "Sequence invalidation or expiration (upstream)"

	case udt.TerminateReasonListenerClosed:
		return "Listener: The listener.Close function was called."
	case udt.TerminateReasonLingerTimerExpired:
		return "Socket: The linger timer expired."
	case udt.TerminateReasonConnectTimeout:
		return "Socket: The connection timed out when sending the initial handshake."
	case udt.TerminateReasonRemoteSentShutdown:
		return "Remote peer sent a shutdown message."
	case udt.TerminateReasonSocketClosed:
		return "Send: Socket closed. Called udtSocket.Close()."
	case udt.TerminateReasonInvalidPacketIDAck:
		return "Send: Invalid packet ID received in ACK message."
	case udt.TerminateReasonInvalidPacketIDNak:
		return "Send: Invalid packet ID received in NAK message."
	case udt.TerminateReasonCorruptPacketNak:
		return "Send: Invalid NAK packet received."
	case udt.TerminateReasonSignal:
		return "Send: Terminate signal. Called udtSocket.Terminate()."
	default:
		return "Unknown."
	}
}

/*
// downloadFile downloads the file from the target peer
func downloadFile(output io.Writer, publicKey *btcec.PublicKey, hash []byte) (data []byte, err error) {
	peer := core.PeerlistLookup(publicKey)
	if peer == nil {
		return nil, errors.New("peer not connected")
	}

	udtConn, _, err := peer.FileTransferRequestUDT(hash, 0, 0)
	if err != nil {
		return nil, err
	}
	defer udtConn.Close()

	fileSize, transferSize, err := core.FileTransferReadHeaderUDT(udtConn)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(output, "* Indicated file size = %d. Target transfer size = %d\n", fileSize, transferSize)

	// read all data
	data = make([]byte, transferSize)
	n, err := udtConn.Read(data)

	fmt.Fprintf(output, "* Read %d bytes (target %d), error: %v\n", n, transferSize, err)

	return data, err
}
*/
