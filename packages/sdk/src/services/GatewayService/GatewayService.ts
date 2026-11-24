import {
  CreateGatewayRequest,
  CreateGatewayResponse,
  DeleteGatewayRequest,
  Gateway,
  GetGatewayRequest,
  GetGatewayResponse,
  GetGatewaysResponse,
  UpdateGatewayRequest,
  UpdateGatewayResponse,
} from "@lwn-simulator/contracts";
import { BaseService } from "../../models";

export class GatewayService extends BaseService {
  constructor(baseUrl: string) {
    super("gateway", baseUrl);
  }

  public createGateway = async (
    req: CreateGatewayRequest,
  ): Promise<Gateway> => {
    const res: CreateGatewayResponse = await this.apiCaller.post(
      "/create-gateway",
      req,
    );
    return res.gateway;
  };

  public getGateway = async (req: GetGatewayRequest): Promise<Gateway> => {
    const res: GetGatewayResponse = await this.apiCaller.get(
      `/get-gateway/${encodeURIComponent(req.id)}`,
    );
    return res.gateway;
  };

  public getGateways = async (): Promise<Array<Gateway>> => {
    const res: GetGatewaysResponse = await this.apiCaller.get("/get-gateways");
    return res.gateways;
  };

  public updateGateway = async (
    req: UpdateGatewayRequest,
  ): Promise<Gateway> => {
    const res: UpdateGatewayResponse = await this.apiCaller.put(
      `/update-gateway/${encodeURIComponent(req.id)}`,
      req,
    );
    return res.gateway;
  };

  public deleteGateway = async (req: DeleteGatewayRequest): Promise<void> => {
    await this.apiCaller.delete(`/delete-gateway/${encodeURIComponent(req.id)}`);
  };
}
