import { describe, expect, it } from "vitest";

import { defaultLanguage, resources, supportedLanguages } from "./resources";

type ResourceLeaf = {
  path: string;
  type: string;
  placeholders: string[];
};

function resourceContract(value: unknown, path = ""): ResourceLeaf[] {
  if (typeof value === "string") {
    return [
      {
        path,
        type: "string",
        placeholders: [...value.matchAll(/{{\s*([^},\s]+)[^}]*}}/g)]
          .map((match) => match[1])
          .sort(),
      },
    ];
  }

  if (Array.isArray(value)) {
    return value.flatMap((item, index) =>
      resourceContract(item, `${path}.${index}`),
    );
  }

  if (value && typeof value === "object") {
    return Object.entries(value).flatMap(([key, item]) =>
      resourceContract(item, path ? `${path}.${key}` : key),
    );
  }

  return [{ path, type: typeof value, placeholders: [] }];
}

describe("translation resources", () => {
  it.each(supportedLanguages)(
    "%s matches the default language resource contract",
    (language) => {
      expect(resourceContract(resources[language])).toEqual(
        resourceContract(resources[defaultLanguage]),
      );
    },
  );
});
