import {
  CreateDeviceRequest,
  CreateDeviceResponse,
  DeleteDeviceRequest,
  Device,
  GetDeviceRequest,
  GetDeviceResponse,
  GetDevicesResponse,
  UpdateDeviceRequest,
  UpdateDeviceResponse,
} from "@lwn-simulator/contracts";
import { BaseService } from "../../models";

/**
 * Service responsible for communicating with the device API.
 *
 * Provides operations for creating, retrieving, updating, and deleting
 * devices through the configured API.
 *
 * The service extends {@link BaseService} and is scoped to a configured API.
 */
export class DeviceService extends BaseService {
  constructor(baseUrl: string) {
    super("device", baseUrl);
  }

  /**
   * Creates a device through the device API.
   *
   * @param req Request containing the device data.
   * @returns The newly created device.
   */
  public createDevice = async (req: CreateDeviceRequest): Promise<Device> => {
    const res: CreateDeviceResponse = await this.apiCaller.post(
      `/create-device`,
      req,
    );

    return res.device;
  };

  /**
   * Retrieves a device by its identifier.
   *
   * @param req Request containing the device identifier.
   * @returns The requested device.
   */
  public getDevice = async (req: GetDeviceRequest): Promise<Device> => {
    const res: GetDeviceResponse = await this.apiCaller.get(
      `/get-device/${encodeURIComponent(req.id)}`,
    );

    return res.device;
  };

  /**
   * Retrieves all devices through the device API.
   *
   * @returns An array containing all available devices.
   */
  public getDevices = async (): Promise<Array<Device>> => {
    const res: GetDevicesResponse = await this.apiCaller.get(`/get-devices`);

    return res.devices;
  };

  /**
   * Updates an existing device.
   *
   * @param req Request containing the device identifier and updated data.
   * @returns The updated device.
   */
  public updateDevice = async (req: UpdateDeviceRequest): Promise<Device> => {
    const res: UpdateDeviceResponse = await this.apiCaller.put(
      `/update-device/${encodeURIComponent(req.id)}`,
      req,
    );

    return res.device;
  };

  /**
   * Deletes an existing device.
   *
   * @param req Request containing the device identifier.
   */
  public deleteDevice = async (req: DeleteDeviceRequest): Promise<void> => {
    await this.apiCaller.delete(
      `/delete-device/${encodeURIComponent(req.id)}`,
    );
  };
}
