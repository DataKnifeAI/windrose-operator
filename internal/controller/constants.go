package controller

const (
	finalizerName = "windrose.dataknife.ai/finalizer"

	defaultServerImage                        = "windroseserver/windroseserver:latest"
	defaultGatewayClassName                   = "envoy"
	defaultDirectConnectionPort         int32 = 7777
	defaultMaxPlayers                   int32 = 4
	defaultStorageSize                        = "35Gi"
	defaultDirectConnectionProxyAddress       = "0.0.0.0"
	defaultP2PProxyAddress                    = "127.0.0.1"

	containerUser = int64(1000)

	savedMountPath             = "/home/ue_user/app/R5/Saved"
	serverDescriptionMountPath = "/home/ue_user/app/R5/ServerDescription.json"
	serverDescriptionConfigKey = "ServerDescription.json"

	gatewayListenerGameTCP = "game-tcp"
	gatewayListenerGameUDP = "game-udp"
)
