package cloudservers

import "github.com/gophercloud/gophercloud"

func resetPwdURL(sc *gophercloud.ServiceClient, serverID string) string {
	return sc.ServiceURL("servers", serverID, "os-reset-password")
}

func changeURL(sc *gophercloud.ServiceClient, serverID string) string {
	return sc.ServiceURL("cloudservers", serverID, "changeos")
}


func reinstallOSURL(sc *gophercloud.ServiceClient, serverID string) string {
	return sc.ServiceURL("cloudservers", serverID, "reinstallos")
}

func resizeFlavorURL(sc *gophercloud.ServiceClient) string {
	return sc.ServiceURL("resize_flavors")
}

func getSecurityGroupURL(sc *gophercloud.ServiceClient, serverID string) string {
	return sc.ServiceURL("servers", serverID,"os-security-groups")
}

func removeSecurityGroupURL(sc *gophercloud.ServiceClient, serverID string)string {
	return sc.ServiceURL("servers", serverID,"action")
}

func addSecurityGroupURL(sc *gophercloud.ServiceClient, serverID string)string {
	return sc.ServiceURL("servers", serverID,"action")
}
