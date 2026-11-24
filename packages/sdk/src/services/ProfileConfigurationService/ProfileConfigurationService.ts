import { GatewayBridgeConfig } from "@lwn-simulator/contracts";
import { BaseService } from "../../models";

export class ProfileConfigurationService extends BaseService {
  constructor(baseUrl: string) {
    super("configuration", baseUrl);
  }

  public getGatewayBridge = async (): Promise<GatewayBridgeConfig> =>
    this.apiCaller.get<GatewayBridgeConfig>("/gateway-bridge");

  public updateGatewayBridge = async (
    config: GatewayBridgeConfig,
  ): Promise<GatewayBridgeConfig> =>
    this.apiCaller.put<GatewayBridgeConfig>("/gateway-bridge", config);
}
