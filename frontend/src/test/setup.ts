import "@testing-library/jest-dom/vitest";
import { cleanup } from "@testing-library/react";
import { afterEach } from "vitest";

afterEach(cleanup);

Object.assign(navigator, {
  clipboard: {
    writeText: () => Promise.resolve(),
  },
});
