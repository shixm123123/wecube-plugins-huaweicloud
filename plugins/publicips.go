package plugins

import (
	"fmt"
	"github.com/gophercloud/gophercloud/openstack/vpc/v1/publicips"
	"github.com/sirupsen/logrus"
	"strconv"
)

const (
	PUBLIC_IP_TYPE_5_BGP       = "5_bgp"
	BANDWIDTH_SHARE_TYPE_PER   = "PER"   //独占带宽
	BANDWIDTH_SHARE_TYPE_WHOLE = "WHOLE" //共享带宽

	BANDWIDTH_SIZE_MAX = 2000
	BANDWIDTH_SIZE_MIN = 1
)

func getPublicIpInfo(params CloudProviderParam, id string) (*publicips.PublicIP, error) {
	sc, err := CreateVpcServiceClientV1(params)
	if err != nil {
		return nil, err
	}

	publicIp, err := publicips.Get(sc, id).Extract()
	if err != nil {
		logrus.Errorf("getPublicIp meet err=%v", err)
	}
	return publicIp, err
}

func createPublicIp(params CloudProviderParam, bandwidthSize string, enterpriseProjectId string) (*publicips.PublicIPCreateResp, error) {
	sc, err := CreateVpcServiceClientV1(params)
	if err != nil {
		return nil, err
	}

	size, _ := strconv.Atoi(bandwidthSize)
	resp, err := publicips.Create(sc, publicips.CreateOpts{
		Publicip: publicips.PublicIPRequest{
			Type:      PUBLIC_IP_TYPE_5_BGP,
			IPVersion: 4,
		},
		Bandwidth: publicips.BandWidth{
			Name:      "wecubeCreated",
			ShareType: BANDWIDTH_SHARE_TYPE_PER,
			Size:      size,
		},
		EnterpriseProjectId: enterpriseProjectId,
	}).Extract()
	if err != nil {
		logrus.Errorf("createPublicIp meet err=%v", err)
	}

	return resp, err
}

func updatePublicIpPortId(params CloudProviderParam, lbId string, portId string) error {
	sc, err := CreateVpcServiceClientV1(params)
	if err != nil {
		return err
	}

	_, err = publicips.Update(sc, lbId, publicips.UpdateOpts{
		PortId: portId,
	}).Extract()
	if err != nil {
		logrus.Errorf("updatePublicIpPortId meet err=%v", err)
	}

	return err
}

func deletePublicIp(params CloudProviderParam, id string) error {
	sc, err := CreateVpcServiceClientV1(params)
	if err != nil {
		return err
	}

	resp := publicips.Delete(sc, id)
	if resp.Err != nil {
		logrus.Errorf("deletePublicIp meet err=%v", err)
		return resp.Err
	}

	return nil
}

func getPublicIpByPortId(params CloudProviderParam, portId string) (*publicips.PublicIP, error) {
	sc, err := CreateVpcServiceClientV1(params)
	if err != nil {
		return nil, err
	}

	allPages, err := publicips.List(sc, publicips.ListOpts{
		Limit: 100,
	}).AllPages()
	if err != nil {
		logrus.Errorf("getPublicIpByPortId list meet err=%v", err)
		return nil, err
	}

	publicipList, err := publicips.ExtractPublicIPs(allPages)
	if err != nil {
		logrus.Errorf("getPublicIpByPortId ExtractPublicIPs meet err=%v", err)
		return nil, err
	}

	for _, resp := range publicipList {
		if resp.PortId == portId {
			return &resp, nil
		}
	}
	return nil, fmt.Errorf("can't found publicIp by portId(%v)", portId)
}
