package constants

const (
	// PortFTP is the FTP port number.
	PortFTP = 21

	// PortSSH is the SSH port number.
	PortSSH = 22

	// PortTelnet is the telnet port number.
	PortTelnet = 23

	// PortHTTP is the HTTP port number.
	PortHTTP = 80

	// PortSNMP is the SNMP port number.
	PortSNMP = 161

	// PortHTTPS is the HTTPS port number.
	PortHTTPS = 443

	// PortNETCONF is the NETCONF (over ssh) port number.
	PortNETCONF = 830

	// PortQemuTelnet is the Qemu management telnet (for vrnetlab generally) port number.
	PortQemuTelnet = 5000

	// PortVNC is the VNC port number.
	PortVNC = 5900

	// PortGNMIArista is the Arista default GNMI port number.
	PortGNMIArista = 6030

	// PortGNMI is the GNMI default port number.
	PortGNMI = 9339

	// PortGRIBI is the GRIBI port number.
	PortGRIBI = 9340

	// PortP4RT is the p4 runtime port number.
	PortP4RT = 9559

	// PortGNMINokia is the Nokia default GNMI port number.
	PortGNMINokia = 57400

	// HealthProbePort is the port number for kubernetes health endpoints to run on.
	HealthProbePort = 8080
)
