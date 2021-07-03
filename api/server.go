package api

import (
	"DawnTorrent/api/torrent_state"
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"net"
)

type CommandServer struct {
	torrent_state.UnimplementedControlTorrentServer
	myapp Control
}

func (s CommandServer) AddTorrent(ctx context.Context, info *torrent_state.TorrentInfo) (*torrent_state.SingleTorrentState, error)  {

	command := command(func() (interface{},error) {
		s.myapp.createNewTorrent(info.Path)
		return errors.New("22")
	})

	res := command.exec()
	return res.(*torrent_state.SingleTorrentState),nil
}

func (s CommandServer) Subscribe(subscription *torrent_state.Subscription, stream torrent_state.ControlTorrent_SubscribeServer) error  {

	s.myapp.subScribeToTorrentState(subscription,torrent_state)

	return nil
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
	torrent_state.RegisterControlTorrentServer(server,CommandServer{})

	err = server.Serve(tcp)
	if err != nil {
		logrus.Fatal(err.Error())
		return
	}

}