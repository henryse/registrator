package etcd

import (
	"github.com/henryse/registrator/bridge"
	"gopkg.in/coreos/go-etcd.v2/etcd"
	"log"
	"net"
	"net/url"
	"strconv"
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

	return &EtcdAdapter{client: etcd.NewClient(urls), path: uri.Path}
}

type EtcdAdapter struct {
	client *etcd.Client
	path   string
}

func (r *EtcdAdapter) Ping() error {
	r.syncEtcdCluster()

	rr := etcd.NewRawRequest("GET", "version", nil, nil)
	_, err := r.client.SendRequest(rr)

	if err != nil {
		return err
	}
	return nil
}

func (r *EtcdAdapter) syncEtcdCluster() {
	var result bool
	result = r.client.SyncCluster()

	if !result {
		log.Println("etcd: sync cluster was unsuccessful")
	}
}

func (r *EtcdAdapter) servicePath(service *bridge.Service) string {
	return r.path + "/" + service.Name + "/" + service.ID
}

func (r *EtcdAdapter) setValue(service *bridge.Service, key string, value string) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/" + key

	_, err := r.client.Set(path, value, uint64(service.TTL))

	if err != nil {
		log.Println("etcd: failed to register service:", err)
	}

	return err
}

func (r *EtcdAdapter) setTags(service *bridge.Service) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/tags"

	var err error
	var returnErr error
	for index, element := range service.Tags {
		_, err = r.client.Set(path+"/"+strconv.Itoa(index), element, uint64(service.TTL))

		if err != nil {
			log.Println("etcd: failed to register service:", err)
			returnErr = err
		}
	}

	return returnErr
}

func (r *EtcdAdapter) setAttrs(service *bridge.Service) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/attrs"

	var returnErr error
	for key, value := range service.Attrs {
		_, err := r.client.Set(path+"/"+key, value, uint64(service.TTL))

		if err != nil {
			log.Println("etcd: failed to register service:", err)
			returnErr = err
		}
	}

	return returnErr
}

func (r *EtcdAdapter) Register(service *bridge.Service) error {
	var err error
	var returnErr error

	err = r.setValue(service, "address", net.JoinHostPort(service.IP, strconv.Itoa(service.Port)))

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "port_type", service.Origin.PortType)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "host_port", service.Origin.HostPort)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "host_ip", service.Origin.HostIP)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "exposed_port", service.Origin.ExposedPort)

	if err != nil {
		returnErr = err
	}

	err = r.setValue(service, "exposed_ip", service.Origin.ExposedIP)

	if err != nil {
		returnErr = err
	}

	err = r.setTags(service)

	if err != nil {
		returnErr = err
	}

	err = r.setAttrs(service)

	return returnErr
}

func (r *EtcdAdapter) Deregister(service *bridge.Service) error {
	r.syncEtcdCluster()

	_, err := r.client.Delete(r.servicePath(service), true)

	if err != nil {
		log.Println("etcd: failed to deregister service:", err)
	}
	return err
}

func (r *EtcdAdapter) Refresh(service *bridge.Service) error {
	return r.Register(service)
}

func (r *EtcdAdapter) Services() ([]*bridge.Service, error) {
	return []*bridge.Service{}, nil
}
