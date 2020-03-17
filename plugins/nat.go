package plugins

import (
	"fmt"
	"github.com/huaweicloud/golangsdk"
	"github.com/huaweicloud/golangsdk/openstack"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/natgateways"
	"github.com/huaweicloud/golangsdk/openstack/networking/v2/extensions/snatrules"
	"github.com/sirupsen/logrus"
	"strings"
)

func createNatServiceClient(params CloudProviderParam) (*golangsdk.ServiceClient, error) {
	if err := isCloudProviderParamValid(params); err != nil {
		return nil, err
	}

	identifyMap, _ := GetMapFromString(params.IdentityParams)
	cloudMap, _ := GetMapFromString(params.CloudParams)
	identityURL := "https://iam." + cloudMap[CLOUD_PARAM_REGION] + "." + cloudMap[CLOUD_PARAM_CLOUD_DOAMIN_NAME] + "/v3"

	opts := golangsdk.AKSKAuthOptions{
		IdentityEndpoint: identityURL,
		AccessKey:        identifyMap[IDENTITY_ACCESS_KEY],
		SecretKey:        identifyMap[IDENTITY_SECRET_KEY],
		//DomainID:         identifyMap[IDENTITY_DOMAIN_ID],
		ProjectId: cloudMap[CLOUD_PARAM_PROJECT_ID],
		Domain:    cloudMap[CLOUD_PARAM_CLOUD_DOAMIN_NAME],
		Region:    cloudMap[CLOUD_PARAM_REGION],
	}
	client, err := openstack.NewClient(identityURL)
	if err != nil {
		logrus.Errorf("new client failed err=%v", err)
		return nil, err
	}
	err = openstack.Authenticate(client, opts)
	if err != nil {
		logrus.Errorf("createNatServiceClient auth failed err=%v", err)
		return nil, err
	}
	sc, err := openstack.NewNatV2(client, golangsdk.EndpointOpts{
		Region: cloudMap[CLOUD_PARAM_REGION],
	})
	if err != nil {
		logrus.Errorf("createNatServiceClient meet err=%v", err)
		return nil, err
	}
	return sc, err
}

var natActions = make(map[string]Action)

func init() {
	natActions["create"] = new(NatCreateAction)
	natActions["delete"] = new(NatDeleteAction)
	natActions["add-snat-rule"] = new(AddSnatRuleAction)
	natActions["delete-snat-rule"] = new(DeleteSnatRuleAction)
}

type NatPlugin struct {
}

func (plugin *NatPlugin) GetActionByName(actionName string) (Action, error) {
	action, found := natActions[actionName]
	if !found {
		logrus.Errorf("natGateway plugin,action = %s not found", actionName)
		return nil, fmt.Errorf("natGateway plugin,action = %s not found", actionName)
	}
	return action, nil
}

type NatCreateInputs struct {
	Inputs []NatCreateInput `json:"inputs,omitempty"`
}

type NatCreateInput struct {
	CallBackParameter
	CloudProviderParam
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`

	VpcId    string `json:"vpc_id,omitempty"`
	SubnetId string `json:"subnet_id,omitempty"`
}

type NatCreateOutputs struct {
	Outputs []NatCreateOutput `json:"outputs,omitempty"`
}

type NatCreateOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type NatCreateAction struct {
}

func (action *NatCreateAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs NatCreateInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func checkNatGatewayCreateParam(input NatCreateInput) error {
	if err := isCloudProviderParamValid(input.CloudProviderParam); err != nil {
		return err
	}

	if input.VpcId == "" {
		return fmt.Errorf("vpcId is empty")
	}

	if input.SubnetId == "" {
		return fmt.Errorf("subnetId is empty")
	}
	return nil
}

func isNatGatewayExist(sc *golangsdk.ServiceClient, id string) (bool, error) {
	_, err := natgateways.Get(sc, id).Extract()
	if err != nil {
		if strings.Contains(err.Error(), "No Nat Gateway exist") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func createNatGateway(input NatCreateInput) (output NatCreateOutput, err error) {
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

	if err = checkNatGatewayCreateParam(input); err != nil {
		return
	}

	sc, err := createNatServiceClient(input.CloudProviderParam)
	if err != nil {
		return
	}

	//check if exist
	if input.Id != "" {
		exist := false
		exist, err = isNatGatewayExist(sc, input.Id)
		if err == nil && exist {
			output.Id = input.Id
			return
		}
	}

	//create natgateway
	cloudMap, _ := GetMapFromString(input.CloudProviderParam.CloudParams)
	opts := natgateways.CreateOpts{
		TenantID:          cloudMap[CLOUD_PARAM_PROJECT_ID],
		Name:              "wecubeCreated",
		Spec:              "1",
		RouterID:          input.VpcId,
		InternalNetworkID: input.SubnetId,
	}

	fmt.Printf("opts=%++v\n", opts)

	result, err := natgateways.Create(sc, opts).Extract()
	if err != nil {
		return
	}
	output.Id = result.ID
	return
}

func (action *NatCreateAction) Do(inputs interface{}) (interface{}, error) {
	gateways, _ := inputs.(NatCreateInputs)
	outputs := NatCreateOutputs{}
	var finalErr error

	for _, input := range gateways.Inputs {
		output, err := createNatGateway(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all natGateway = %v are created", gateways)
	return &outputs, finalErr
}

//---------delete nat gateway------------//
type NatDeleteInputs struct {
	Inputs []NatDeleteInput `json:"inputs,omitempty"`
}

type NatDeleteInput struct {
	CallBackParameter
	CloudProviderParam
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type NatDeleteOutputs struct {
	Outputs []NatDeleteOutput `json:"outputs,omitempty"`
}

type NatDeleteOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type NatDeleteAction struct {
}

func (action *NatDeleteAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs NatDeleteInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func deleteNatGateway(input NatDeleteInput) (output NatDeleteOutput, err error) {
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

	if err = isCloudProviderParamValid(input.CloudProviderParam); err != nil {
		return
	}
	if input.Id == "" {
		err = fmt.Errorf("empty id")
		return
	}

	sc, err := createNatServiceClient(input.CloudProviderParam)
	if err != nil {
		return
	}

	exist, err := isNatGatewayExist(sc, input.Id)
	if err != nil || !exist {
		return
	}

	if err = natgateways.Delete(sc, input.Id).ExtractErr(); err != nil {
		logrus.Errorf("natgateway(%v) delete failed,err=%v", input.Id, err)
	}

	return
}

func (action *NatDeleteAction) Do(inputs interface{}) (interface{}, error) {
	gateways, _ := inputs.(NatDeleteInputs)
	outputs := NatDeleteOutputs{}
	var finalErr error

	for _, input := range gateways.Inputs {
		output, err := deleteNatGateway(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all natGateway = %v are created", gateways)
	return &outputs, finalErr
}

//------------add snat rule----------------//
type AddSnatRuleInputs struct {
	Inputs []AddSnatRuleInput `json:"inputs,omitempty"`
}

type AddSnatRuleInput struct {
	CallBackParameter
	CloudProviderParam
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`

	GatewayId  string `json:"gateway_id,omitempty"`
	SubnetId   string `json:"subnet_id,omitempty"`
	PublicIpId string `json:"public_ip_id,omitempty"`
}

type AddSnatRuleOutputs struct {
	Outputs []AddSnatRuleOutput `json:"outputs,omitempty"`
}

type AddSnatRuleOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type AddSnatRuleAction struct {
}

func (action *AddSnatRuleAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs AddSnatRuleInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func isSnatRuleExist(sc *golangsdk.ServiceClient, id string) (bool, error) {
	_, err := snatrules.Get(sc, id).Extract()
	if err != nil {
		if strings.Contains(err.Error(), "No Snat Rule exist") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func checkAddSnatParam(input AddSnatRuleInput) error {
	if err := isCloudProviderParamValid(input.CloudProviderParam); err != nil {
		return err
	}

	if input.SubnetId == "" {
		return fmt.Errorf("subnetId is empty")
	}
	if input.GatewayId == "" {
		return fmt.Errorf("gatewayId is empty")
	}
	if input.PublicIpId == "" {
		return fmt.Errorf("publicIpId is empty")
	}

	return nil
}

func addSnatRule(input AddSnatRuleInput) (output AddSnatRuleOutput, err error) {
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

	if err = checkAddSnatParam(input); err != nil {
		return
	}

	sc, err := createNatServiceClient(input.CloudProviderParam)
	if err != nil {
		return
	}

	if input.Id != "" {
		exist := false
		exist, err = isSnatRuleExist(sc, input.Id)
		if err == nil && exist {
			output.Id = input.Id
			return
		}
	}

	opts := snatrules.CreateOpts{
		NatGatewayID: input.GatewayId,
		NetworkID:    input.SubnetId,
		FloatingIPID: input.PublicIpId,
		SourceType:   0,
	}
	resp, err := snatrules.Create(sc, opts).Extract()
	if err != nil {
		logrus.Errorf("create snat rule failed,err=%v", err)
		return
	}

	output.Id = resp.ID
	return
}

func (action *AddSnatRuleAction) Do(inputs interface{}) (interface{}, error) {
	rules, _ := inputs.(AddSnatRuleInputs)
	outputs := AddSnatRuleOutputs{}
	var finalErr error

	for _, input := range rules.Inputs {
		output, err := addSnatRule(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all snat rule = %v are created", rules)
	return &outputs, finalErr
}

//------------delete snat rule----------------//

type DeleteSnatRuleInputs struct {
	Inputs []DeleteSnatRuleInput `json:"inputs,omitempty"`
}

type DeleteSnatRuleInput struct {
	CallBackParameter
	CloudProviderParam
	Guid string `json:"guid,omitempty"`
	Id   string `json:"id,omitempty"`
}

type DeleteSnatRuleOutputs struct {
	Outputs []DeleteSnatRuleOutput `json:"outputs,omitempty"`
}

type DeleteSnatRuleOutput struct {
	CallBackParameter
	Result
	Guid string `json:"guid,omitempty"`
}

type DeleteSnatRuleAction struct {
}

func (action *DeleteSnatRuleAction) ReadParam(param interface{}) (interface{}, error) {
	var inputs DeleteSnatRuleInputs
	err := UnmarshalJson(param, &inputs)
	if err != nil {
		return nil, err
	}
	return inputs, nil
}

func deleteSnatRule(input DeleteSnatRuleInput) (output DeleteSnatRuleOutput, err error) {
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

	if err = isCloudProviderParamValid(input.CloudProviderParam); err != nil {
		return
	}
	if input.Id == "" {
		err = fmt.Errorf("empty id")
		return
	}

	sc, err := createNatServiceClient(input.CloudProviderParam)
	if err != nil {
		return
	}

	exist, err := isSnatRuleExist(sc, input.Id)
	if err != nil || !exist {
		return
	}

	if err = snatrules.Delete(sc, input.Id).ExtractErr(); err != nil {
		logrus.Errorf("delete snat rule failed err=%v", err)
	}

	return
}

func (action *DeleteSnatRuleAction) Do(inputs interface{}) (interface{}, error) {
	rules, _ := inputs.(DeleteSnatRuleInputs)
	outputs := DeleteSnatRuleOutputs{}
	var finalErr error

	for _, input := range rules.Inputs {
		output, err := deleteSnatRule(input)
		if err != nil {
			finalErr = err
		}
		outputs.Outputs = append(outputs.Outputs, output)
	}

	logrus.Infof("all snat rule = %v are delete", rules)
	return &outputs, finalErr
}