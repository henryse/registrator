package etcd

import (
	"log"
	"net"
	"net/url"
	"strconv"

	"github.com/henryse/registrator/bridge"
	"gopkg.in/coreos/go-etcd.v2/etcd"
)

func init() {
	bridge.Register(new(Factory), "etcd")
}

type Factory struct{}

func (f *Factory) New(uri *url.URL) bridge.RegistryAdapter {
	urls := make([]string, 0)
	if uri.Host != "" {
		urls = append(urls, "http://"+uri.Host)
	} else {
		urls = append(urls, "http://127.0.0.1:2379")
	}

	return &Adapter{client: etcd.NewClient(urls), path: uri.Path}
}

type Adapter struct {
	client *etcd.Client

	path string
}

func (r *Adapter) Ping() error {
	r.syncEtcdCluster()

	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := r.client.SendRequest(rr)

	if err != nil {
		return err
	}
	return nil
}

func (r *Adapter) syncEtcdCluster() {
	var result bool
	result = r.client.SyncCluster()

	if !result {
		log.Println("etcd: sync cluster was unsuccessful")
	}
}

func (r *Adapter) Register(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID
	port := strconv.Itoa(service.Port)
	addr := net.JoinHostPort(service.IP, port)

	var err error
	_, err = r.client.Set(path, addr, uint64(service.TTL))

	if err != nil {
		log.Println("etcd: failed to register service:", err)
	}
	return err
}

func (r *Adapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()

	path := r.path + "/" + service.Name + "/" + service.ID

	var err error
	_, err = r.client.Delete(path, false)

	if err != nil {
		log.Println("etcd: failed to deregister service:", err)
	}
	return err
}

func (r *Adapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *Adapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
