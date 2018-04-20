package etcd

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	etcd2 "github.com/coreos/go-etcd/etcd"
	"github.com/henryse/registrator/bridge"
	"gopkg.in/coreos/go-etcd.v0/etcd"
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

	res, err := http.Get(urls[0] + "/version")
	if err != nil {
		log.Fatal("etcd: error retrieving version", err)
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	if match, _ := regexp.Match("0\\.4\\.*", body); match == true {
		log.Println("etcd: using v0 client")
		return &EtcdAdapter{client: etcd.NewClient(urls), path: uri.Path}
	}

	return &EtcdAdapter{client2: etcd2.NewClient(urls), path: uri.Path}
}

type EtcdAdapter struct {
	client  *etcd.Client
	client2 *etcd2.Client

	path string
}

func (r *EtcdAdapter) Ping() error {
	r.syncEtcdCluster()

	var err error
	if r.client != nil {
		rr := etcd.NewRawRequest("GET", "version", nil, nil)
		_, err = r.client.SendRequest(rr)
	} else {
		rr := etcd2.NewRawRequest("GET", "version", nil, nil)
		_, err = r.client2.SendRequest(rr)
	}

	if err != nil {
		return err
	}
	return nil
}

func (r *EtcdAdapter) syncEtcdCluster() {
	var result bool
	if r.client != nil {
		result = r.client.SyncCluster()
	} else {
		result = r.client2.SyncCluster()
	}

	if !result {
		log.Println("etcd: sync cluster was unsuccessful")
	}
}

func (r *EtcdAdapter) servicePath(service *bridge.Service) string {
	return  r.path + "/" + service.Name +"/services/" + service.ID
}

func (r *EtcdAdapter) setValue(service *bridge.Service, key string, value string) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/" + key

	var err error
	if r.client != nil {
		_, err = r.client.Set(path, value, uint64(service.TTL))
	} else {
		_, err = r.client2.Set(path, value, uint64(service.TTL))
	}

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
		if r.client != nil {
			_, err = r.client.Set(path + "/" + strconv.Itoa(index), element, uint64(service.TTL))
		} else {
			_, err = r.client2.Set(path + "/" + strconv.Itoa(index), element, uint64(service.TTL))
		}

		if err != nil {
			log.Println("etcd: failed to register service:", err)
			returnErr = err
		}
	}

	return returnErr
}

func (r *EtcdAdapter) setAttrs(service *bridge.Service) error {
	r.syncEtcdCluster()
	path := r.servicePath(service) + "/Attrs"

	var err error
	var returnErr error
	for key, value := range service.Attrs {
		if r.client != nil {
			_, err = r.client.Set(path + "/" + key, value, uint64(service.TTL))
		} else {
			_, err = r.client2.Set(path + "/" + key, value, uint64(service.TTL))
		}

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

	var err error
	if r.client != nil {
		_, err = r.client.Delete(r.servicePath(service), true)
	} else {
		_, err = r.client2.Delete(r.servicePath(service), true)
	}

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

