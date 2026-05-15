package matchmaking_handlers

import (
	"github.com/gorilla/websocket"
	"github.com/remixfn/xenon/modules/matchmaking/types"
)

func GetAllClients() []*websocket.Conn {
	types.ClientM.RLock()
	defer types.ClientM.RUnlock()

	var connList []*websocket.Conn
	for client := range types.Clients {
		connList = append(connList, client.Conn)
	}
	return connList
}

func GetAllClientsViaData(version string, playlist string, region string) []*types.Client {
	types.ClientM.RLock()
	defer types.ClientM.RUnlock()

	var connList []*types.Client
	for client := range types.Clients {
		if client.Payload.Version == version && client.Payload.Playlist == playlist && client.Payload.Region == region {
			connList = append(connList, client)
		}
	}

	return connList
}

func GetAllClientsViaDataLen(version string, playlist string, region string) int {
	types.ClientM.RLock()
	defer types.ClientM.RUnlock()

	var connList []*types.Client
	for client := range types.Clients {
		if client.Payload.Version == version && client.Payload.Playlist == playlist && client.Payload.Region == region {
			connList = append(connList, client)
		}
	}

	return len(connList)
}
