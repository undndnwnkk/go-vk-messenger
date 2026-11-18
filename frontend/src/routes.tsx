import { Navigate, type RouteObject } from "react-router";

import { GuestOnly, RequireAuth } from "./auth/RouteGuards";
import { LoginPage } from "./pages/auth/LoginPage";
import { RegisterPage } from "./pages/auth/RegisterPage";
import { ChatPage } from "./pages/chats/ChatPage";
import { ChatsLayout } from "./pages/chats/ChatsLayout";
import { NoChatSelected } from "./pages/chats/NoChatSelected";
import { NotFoundPage } from "./pages/NotFoundPage";

export const routes: RouteObject[] = [
  { path: "/", element: <Navigate to="/chats" replace /> },
  {
    element: <GuestOnly />,
    children: [
      { path: "/login", element: <LoginPage /> },
      { path: "/register", element: <RegisterPage /> },
    ],
  },
  {
    element: <RequireAuth />,
    children: [
      {
        path: "/chats",
        element: <ChatsLayout />,
        children: [
          { index: true, element: <NoChatSelected /> },
          { path: ":id", element: <ChatPage /> },
        ],
      },
    ],
  },
  { path: "*", element: <NotFoundPage /> },
];
