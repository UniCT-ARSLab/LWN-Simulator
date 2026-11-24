import { describe, expect, it } from "vitest";
import { hasPayloadDecoder, isValidJavaScript } from "./payloadCodec";

describe("isValidJavaScript", () => {
  it("accepts a valid payload decoder", () => {
    expect(
      isValidJavaScript(
        "function decodeUplink(input) { return { data: input.bytes }; }",
      ),
    ).toBe(true);
  });

  it("rejects an invalid payload decoder", () => {
    expect(
      isValidJavaScript("function decodeUplink(input) { return { data: }; }"),
    ).toBe(false);
  });
});

describe("hasPayloadDecoder", () => {
  it.each([
    "function decodeUplink(input) { return { data: {} }; }",
    "const decodeUplink = input => ({ data: {} });",
    "let decodeUplink = function(input) { return { data: {} }; };",
  ])("accepts a synchronous top-level decoder: %s", (script) => {
    expect(hasPayloadDecoder(script)).toBe(true);
  });

  it.each([
    "// function decodeUplink(input) {}\nconst other = 1;",
    'const description = "function decodeUplink(input) {}";',
    "function wrapper() { function decodeUplink(input) {} }",
    "const decodeUplink = 42;",
    "async function decodeUplink(input) { return { data: {} }; }",
    "const decodeUplink = async input => ({ data: {} });",
    "function* decodeUplink(input) { yield input; }",
    "function decodeUplink( {",
  ])("rejects an unavailable or asynchronous decoder: %s", (script) => {
    expect(hasPayloadDecoder(script)).toBe(false);
  });
});
