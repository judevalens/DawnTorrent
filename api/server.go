package api

import (
	"DawnTorrent/rpc/torrent_state"
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net"
	"strconv"
)

const (
	INTERNAL = 13
)

type CommandServer struct {
	torrent_state.UnimplementedStateServiceServer
	myapp Control
}

func (s CommandServer) AddTorrent(ctx context.Context, info *torrent_state.TorrentInfo) (*torrent_state.TorrentState, error)  {

	command := command(func() (interface{},error) {
		err := s.myapp.createNewTorrent(info.Path)
		if err != nil {
			return nil, err
		}
		return errors.New("22"), nil
	})

	res, _ := command.exec()
	return res.(*torrent_state.TorrentState),nil
}

func (s CommandServer) Subscribe(subscription *torrent_state.Subscription, stream torrent_state.StateService_SubscribeServer) error  {

	err := s.myapp.subScribeToTorrentState(subscription, stream)
	if err != nil {
		err := stream.SetHeader(map[string][]string{
			"error_code":    {strconv.Itoa(INTERNAL)},
			"error_message": {err.Error()},
		})
		if err != nil {
			return err
		}
		return err
	}

	return nil
}


func (s CommandServer) RemoveTorrent(context.Context, *torrent_state.TorrentInfo) (*torrent_state.TorrentState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method RemoveTorrent not implemented")
}
func (s CommandServer) ControlTorrent(context.Context, *torrent_state.Control) (*torrent_state.TorrentState, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ControlTorrent not implemented")
}

func StartServer(){
	addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%v", 22292))
	if err != nil {
		logrus.Fatal(err)
		return
	}

	tcp, err := net.ListenTCP("tcp",addr)
	if err != nil {
		return
	}

	server := grpc.NewServer(grpc.EmptyServerOption{})
	torrent_state.RegisterStateServiceServer(server, CommandServer{})

	err = server.Serve(tcp)
	if err != nil {
		logrus.Fatal(err.Error())
		return
	}

}