import { screen } from "@testing-library/react";
import { http, HttpResponse } from "msw";

import { API_URL } from "../api/client";
import { server } from "../test/server";
import { renderApp } from "../test/renderApp";
import { readToken, writeToken } from "./tokenStore";

const invalidToken = HttpResponse.json(
  { error: { code: "invalid_token", message: "invalid token" } },
  { status: 401 },
);

test("signs the user out and opens the login page when the API answers 401", async () => {
  writeToken("expired-jwt", true);
  const seenAuthorization: (string | null)[] = [];
  server.use(
    http.get(`${API_URL}/api/v1/me`, ({ request }) => {
      seenAuthorization.push(request.headers.get("Authorization"));
      return invalidToken.clone();
    }),
    http.get(`${API_URL}/api/v1/chats`, () => invalidToken.clone()),
  );

  const { router } = renderApp("/chats");

  expect(await screen.findByRole("heading", { name: "Welcome back" })).toBeInTheDocument();
  expect(router.state.location.pathname).toBe("/login");
  expect(readToken()).toBeNull();
  expect(seenAuthorization).toContain("Bearer expired-jwt");
});
