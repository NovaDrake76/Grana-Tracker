import { createSystem, defaultConfig, defineConfig } from "@chakra-ui/react";

const config = defineConfig({
  theme: {
    tokens: {
      colors: {
        brand: {
          50: { value: "#e8f4fd" },
          100: { value: "#bee3f8" },
          200: { value: "#90cdf4" },
          300: { value: "#63b3ed" },
          400: { value: "#4299e1" },
          500: { value: "#0ea5e9" },
          600: { value: "#0284c7" },
          700: { value: "#0369a1" },
          800: { value: "#075985" },
          900: { value: "#0c4a6e" },
        },
        gain: { value: "#22c55e" },
        loss: { value: "#ef4444" },
      },
      fonts: {
        body: {
          value:
            "'Inter', system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        },
        heading: {
          value:
            "'Inter', system-ui, -apple-system, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif",
        },
      },
    },
  },
});

// merging with defaultConfig is mandatory: createSystem(config) alone wipes
// every base token (spacing, radii, fonts) and Chakra components render naked.
export const system = createSystem(defaultConfig, config);
