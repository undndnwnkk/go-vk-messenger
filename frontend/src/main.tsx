import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router";

import { createQueryClient } from "./api/queryClient";
import { AppProviders } from "./AppProviders";
import { routes } from "./routes";
import "./styles/index.css";

const container = document.getElementById("root");
if (container === null) {
  throw new Error("Root element #root is missing");
}

const router = createBrowserRouter(routes);
const queryClient = createQueryClient();

createRoot(container).render(
  <StrictMode>
    <AppProviders queryClient={queryClient}>
      <RouterProvider router={router} />
    </AppProviders>
  </StrictMode>,
);
