package plugins

import (
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/vpc/v1/subnets"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

var subnetActions = make(map[string]Action)

func init() {
	subnetActions["create"] = new(SubnetCreateAction)
	subnetActions["delete"] = new(SubnetDeleteAction)
}

type SubnetPlugin struct {
}

func (plugin *SubnetPlugin) GetActionByName(actionName string) (Action, error) {
	action, found := subnetActions[actionName]
	if !found {
		logrus.Errorf("subnet plugin,action = %s not found", actionName)
		return nil, fmt.Errorf("VPC plugin,action = %s not found", actionName)
	}
	return action, nil
}

type SubnetCreateInputs struct {
	Inputs []SubnetCreateInput `json:"inputs,omitempty"`
}

type SubnetCreateInput struct {
	CallBackParameter
	CloudProviderParam
	Guid  string `json:"guid,omitempty"`
	Id    string `json:"id,omitempty"`
	VpcId string `json:"vpc_id,omitempty"`
	Name  string `json:"name,omitempty"`
	Cidr  string `json:"cidr,omitempty"`
}

type SubnetCreateOutputs struct {
	Outputs []SubnetCreateOutput `json:"outputs,omitempty"`
}

type SubnetCreateOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type SubnetCreateAction struct {
}

func (action *SubnetCreateAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs SubnetCreateInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func checkCreateSubnetInput(input SubnetCreateInput) error {
	if err := isCloudProvicerParamValid(input.CloudProviderParam); err != nil {
		return err
	}
	if input.VpcId == "" {
		return fmt.Errorf("vpcId is empty")
	}
	if input.Name == "" {
		return fmt.Errorf("name is empty")
	}
	if input.Cidr == "" {
		return fmt.Errorf("cidr is empty")
	}

	if err := isValidCidr(input.Cidr); err != nil {
		return err
	}

	return nil
}

func getSubnetStatus(sc *gophercloud.ServiceClient, subnetId string) (string, error) {
	resp, err := subnets.Get(sc, subnetId).Extract()
	if err != nil {
		return "", err
	}
	return resp.Status, nil
}

func isSubnetExist(sc *gophercloud.ServiceClient, subnetId string) (bool, error) {
	_, err := subnets.Get(sc, subnetId).Extract()
	if err != nil {
		if ue, ok := err.(*gophercloud.UnifiedError); ok {
			if strings.Contains(ue.Message(), "the subnet could not be found") {
				return false, nil
			}
		}
		return false, err
	}
	return true, nil
}

func waitSubnetCreateOk(sc *gophercloud.ServiceClient, subnetId string) error {
	count := 1

	for {
		status, err := getSubnetStatus(sc, subnetId)
		if err != nil {
			return err
		}
		if status == "ACTIVE" {
			return nil
		}
		if status == "ERROR" {
			return fmt.Errorf("create subnet status is ERROR")
		}

		if count > 20 {
			logrus.Errorf("waitSubnetCreateOk is timeout,last status =%v", status)
			return fmt.Errorf("waitSubnetCreateOk is timeout,last status =%v", status)
		}
		time.Sleep(time.Second * 5)
		count++
	}
}

func createSubnet(input SubnetCreateInput) (output SubnetCreateOutput, err error) {
	defer func() {
		output.Guid = input.Guid
		output.CallBackParameter.Parameter = input.CallBackParameter.Parameter
		if err == nil {
			output.Result.Code = RESULT_CODE_SUCCESS
		} else {
			output.Result.Code = RESULT_CODE_ERROR
			output.Result.Message = err.Error()
		}
	}()

	if err = checkCreateSubnetInput(input); err != nil {
		return
	}

	sc, err := CreateVpcServiceClientV1(input.CloudProviderParam)
	if err != nil {
		logrus.Errorf("CreateVpcServiceClient[%v] meet error=%v", VPC_SERVICE_CLIENT_V1, err)
		return
	}

	// check if subnet id exist
	if input.Id != "" {
		exist, subnetExistErr := isSubnetExist(sc, input.Id)
		if subnetExistErr != nil {
			err = subnetExistErr
			return
		}
		if exist {
			output.Id = input.Id
			return
		}
	}

	gatewayIp, err := getCidrGatewayIp(input.Cidr)
	if err != nil {
		return
	}

	resp, err := subnets.Create(sc, subnets.CreateOpts{
		Name:      input.Name,
		Cidr:      input.Cidr,
		GatewayIP: gatewayIp,
		VpcID:     input.VpcId,
	}).Extract()
	if err != nil {
		logrus.Errorf("create subnet meet error=%v", err)
		return
	}

	output.Id = resp.ID
	if err = waitSubnetCreateOk(sc, output.Id); err != nil {
		logrus.Errorf("waitSubnetCreateOk meet err=%v", err)
	}
	return
}

func (action *SubnetCreateAction) Do(inputs interface{}) (interface{}, error) {
	subnets, _ := inputs.(SubnetCreateInputs)
	outputs := SubnetCreateOutputs{}
	var finalErr error

	for _, input := range subnets.Inputs {
		output, err := createSubnet(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all subnets = %v are created", subnets)
	return &outputs, finalErr
}

type SubnetDeleteInputs struct {
	Inputs []SubnetDeleteInput `json:"inputs,omitempty"`
}

type SubnetDeleteInput struct {
	CallBackParameter
	CloudProviderParam
	Guid  string `json:"guid,omitempty"`
	Id    string `json:"id,omitempty"`
	VpcId string `json:"vpc_id,omitempty"`
}

type SubnetDeleteOutputs struct {
	Outputs []SubnetDeleteOutput `json:"outputs,omitempty"`
}

type SubnetDeleteOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
}

type SubnetDeleteAction struct {
}

func (action *SubnetDeleteAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs SubnetDeleteInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func checkDeleteSubnetInput(input SubnetDeleteInput) error {
	if err := isCloudProvicerParamValid(input.CloudProviderParam); err != nil {
		return err
	}
	if input.VpcId == "" {
		return fmt.Errorf("vpcId is empty")
	}
	if input.Id == "" {
		return fmt.Errorf("subnetId is empty")
	}
	return nil
}

func deleteSubnet(input SubnetDeleteInput) (output SubnetDeleteOutput, err error) {
	defer func() {
		output.Guid = input.Guid
		output.CallBackParameter.Parameter = input.CallBackParameter.Parameter
		if err == nil {
			output.Result.Code = RESULT_CODE_SUCCESS
		} else {
			output.Result.Code = RESULT_CODE_ERROR
			output.Result.Message = err.Error()
		}
	}()

	if err = checkDeleteSubnetInput(input); err != nil {
		return
	}

	sc, err := CreateVpcServiceClientV1(input.CloudProviderParam)
	if err != nil {
		logrus.Errorf("CreateVpcServiceClient[%v] meet error=%v", VPC_SERVICE_CLIENT_V1, err)
		return
	}

	//check if subnet exist
	exist, err := isSubnetExist(sc, input.Id)
	if err != nil || !exist {
		return
	}

	resp := subnets.Delete(sc, input.VpcId, input.Id)
	if resp.Err != nil {
		err = resp.Err
		logrus.Errorf("Delete subnet[subnetId=%v] failed, error=%v", input.Id, err)
		return
	}
	return
}

func (action *SubnetDeleteAction) Do(inputs interface{}) (interface{}, error) {
	subnets, _ := inputs.(SubnetDeleteInputs)
	outputs := SubnetDeleteOutputs{}
	var finalErr error

	for _, input := range subnets.Inputs {
		output, err := deleteSubnet(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all subnets = %v are delete", subnets)
	return &outputs, finalErr
}
