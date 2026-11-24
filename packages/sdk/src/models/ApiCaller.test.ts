import { describe, expect, it } from "vitest";
import { ApiCaller } from "./ApiCaller";

describe("ApiCaller", () => {
  it("joins base URLs without duplicating slashes", () => {
    expect(ApiCaller.joinUrl("https://api.example.test///", "/status")).toBe(
      "https://api.example.test/status",
    );
  });

  it("builds service-scoped URLs with encoded paths preserved", () => {
    const caller = new ApiCaller("device", "http://localhost:8000/api/");

    expect(caller.buildUrl("/get-device/device-1")).toBe(
      "http://localhost:8000/api/device/get-device/device-1",
    );
  });
});
