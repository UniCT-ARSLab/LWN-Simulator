import { parse } from "acorn";

export const isValidJavaScript = (value: string): boolean => {
  try {
    parse(value, { ecmaVersion: "latest", sourceType: "script" });
    return true;
  } catch {
    return false;
  }
};

/** Checks for a synchronous decoder declared at script scope without executing it. */
export const hasPayloadDecoder = (value: string): boolean => {
  try {
    const program = parse(value, { ecmaVersion: "latest", sourceType: "script" });
    return program.body.some((statement) => {
      if (statement.type === "FunctionDeclaration") {
        return statement.id?.name === "decodeUplink" && !statement.async && !statement.generator;
      }
      if (statement.type !== "VariableDeclaration") return false;
      return statement.declarations.some((declaration) => {
        const functionNode = declaration.init;
        return declaration.id.type === "Identifier" &&
          declaration.id.name === "decodeUplink" &&
          functionNode != null &&
          (functionNode.type === "FunctionExpression" || functionNode.type === "ArrowFunctionExpression") &&
          !functionNode.async && !functionNode.generator;
      });
    });
  } catch {
    return false;
  }
};
