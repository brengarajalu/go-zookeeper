package zk

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
)

var (
	requests     = make(map[int32]int32) // Map of Xid -> Opcode
	requestsLock = &sync.Mutex{}
)

func trace(conn1, conn2 net.Conn, client bool) {
	defer conn1.Close()
	defer conn2.Close()
	buf := make([]byte, 10*1024)
	init := true
	for {
		_, err := io.ReadFull(conn1, buf[:4])
		if err != nil {
			fmt.Println("1>", client, err)
			return
		}

		blen := int(binary.BigEndian.Uint32(buf[:4]))

		_, err = io.ReadFull(conn1, buf[4:4+blen])
		if err != nil {
			fmt.Println("2>", client, err)
			return
		}

		var cr interface{} = nil
		var opcode int32 = -1
		if client {
			if init {
				cr = &connectRequest{}
			} else {
				xid := int32(binary.BigEndian.Uint32(buf[4:8]))
				opcode = int32(binary.BigEndian.Uint32(buf[8:12]))
				requestsLock.Lock()
				requests[xid] = opcode
				requestsLock.Unlock()
				switch opcode {
				default:
					fmt.Printf("Unknown opcode %d\n", opcode)
				case opClose:
					cr = &closeRequest{}
				case opCreate:
					cr = &createRequest{}
				case opDelete:
					cr = &deleteRequest{}
				case opExists:
					cr = &existsRequest{}
				case opGetAcl:
					cr = &getAclRequest{}
				case opGetChildren:
					cr = &getChildrenRequest{}
				case opGetChildren2:
					cr = &getChildren2Request{}
				case opGetData:
					cr = &getDataRequest{}
				case opPing:
					cr = &pingRequest{}
				case opSetAcl:
					cr = &setAclRequest{}
				case opSetData:
					cr = &setDataRequest{}
				case opSetWatches:
					cr = &setWatchesRequest{}
				case opSync:
					cr = &syncRequest{}
				}
			}
		} else {
			if init {
				cr = &connectResponse{}
			} else {
				xid := int32(binary.BigEndian.Uint32(buf[4:8]))
				zxid := int64(binary.BigEndian.Uint64(buf[8:16]))
				errnum := int32(binary.BigEndian.Uint32(buf[16:20]))
				if xid != -1 || zxid != -1 {
					requestsLock.Lock()
					found := false
					opcode, found = requests[xid]
					if !found {
						println("WEFWEFEW")
						opcode = 0
					}
					delete(requests, xid)
					requestsLock.Unlock()
				} else {
					opcode = opWatcherEvent
				}
				switch opcode {
				default:
					fmt.Printf("Unknown opcode %d\n", opcode)
				case opClose:
					cr = &closeResponse{}
				case opCreate:
					cr = &createResponse{}
				case opDelete:
					cr = &deleteResponse{}
				case opExists:
					cr = &existsResponse{}
				case opGetAcl:
					cr = &getAclResponse{}
				case opGetChildren:
					cr = &getChildrenResponse{}
				case opGetChildren2:
					cr = &getChildren2Response{}
				case opGetData:
					cr = &getDataResponse{}
				case opPing:
					cr = &pingResponse{}
				case opSetAcl:
					cr = &setAclResponse{}
				case opSetData:
					cr = &setDataResponse{}
				case opSetWatches:
					cr = &setWatchesResponse{}
				case opSync:
					cr = &syncResponse{}
				case opWatcherEvent:
					cr = &watcherEvent{}
				}
				if errnum != 0 {
					cr = &responseHeader{}
				}
			}
		}
		opname := "."
		if opcode != -1 {
			opname = opNames[opcode]
		}
		if cr == nil {
			fmt.Printf("%+v %s %+v\n", client, opname, buf[4:4+blen])
		} else {
			if _, err := decodePacket(buf[4:4+blen], cr); err != nil {
				fmt.Println(err)
			}
			fmt.Printf("%+v %s %+v\n", client, opname, cr)
		}

		init = false

		written, err := conn2.Write(buf[:4+blen])
		if err != nil {
			fmt.Println("3>", client, err)
			return
		} else if written != 4+blen {
			fmt.Printf("Written != read: %d != %d\n", written, blen)
			return
		}
	}
}

func handleConnection(conn net.Conn) {
	zkConn, err := net.Dial("tcp", "127.0.0.1:2181")
	if err != nil {
		fmt.Println(err)
		return
	}
	go trace(conn, zkConn, true)
	trace(zkConn, conn, false)
}

func startTracer() {
	ln, err := net.Listen("tcp", "127.0.0.1:2182")
	if err != nil {
		panic(err)
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}
