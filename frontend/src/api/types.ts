import type { components } from "./schema.gen";

type Schemas = components["schemas"];

export type User = Schemas["User"];
export type Chat = Schemas["Chat"];
export type Message = Schemas["Message"];
export type MessagePage = Schemas["MessagePage"];
export type Credentials = Schemas["Credentials"];
export type TokenResponse = Schemas["TokenResponse"];
export type ErrorResponse = Schemas["Error"];
export type UserSummary = Schemas["UserSummary"];
export type ChatLastMessage = Schemas["ChatLastMessage"];
