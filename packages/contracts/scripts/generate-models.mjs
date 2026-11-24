import fs from "fs";

const input = "generated/openapi.ts";
const output = "generated/models.ts";

const content = fs.readFileSync(input, "utf8");

const toPascalCase = (str) =>
    str
    .split(/[^a-zA-Z0-9]+/)
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join("");

const schemasSection = content.match(
  /export interface components \{[\s\S]*?schemas: \{([\s\S]*?)\n    \};/,
);

if (!schemasSection) {
  throw new Error("Schemas section not found");
}

const schemasContent = schemasSection[1];

let result = "";

for (const schema of schemasContent.matchAll(
  /^\s{8}(\w+): Record<string, never>;\s*$/gm,
)) {
  const name = schema[1].charAt(0).toUpperCase() + schema[1].slice(1);

  result += `export interface ${name} {}\n\n`;
}

for (const schema of schemasContent.matchAll(
  /^\s{8}(\w+):\s*((?:"[^"]+"\s*(?:\|\s*)?)+);\s*$/gm,
)) {
  const name = schema[1].charAt(0).toUpperCase() + schema[1].slice(1);
  const values = [...schema[2].matchAll(/"([^"]+)"/g)].map((match) => match[1]);

  result += `export enum ${name} {\n`;

  for (const value of values) {
    result += `  ${toPascalCase(value)} = "${value}",\n`;
  }

  result += `}\n\n`;
}

for (const schema of schemasContent.matchAll(
  /^\s{8}(\w+): \{([\s\S]*?)^\s{8}\};/gm,
)) {
  const name = schema[1].charAt(0).toUpperCase() + schema[1].slice(1);

  let body = schema[2];

  body = body.replace(/components\["schemas"\]\["(\w+)"\]/g, "$1");

  body = body
    .split("\n")
    .map((line) => line.replace(/^ {8}/, ""))
    .join("\n");

  result += `export interface ${name} {${body}}\n\n`;
}

fs.writeFileSync(output, result);

console.log(`Generated ${output}`);
