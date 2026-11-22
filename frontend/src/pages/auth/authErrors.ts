import { errorMessage, isApiError } from "../../api/errors";

export interface AuthFormErrors {
  username?: string;
  password?: string;
  form?: string;
}

const usernameCodes = new Set(["user_exists", "invalid_username"]);
const passwordCodes = new Set(["invalid_password"]);

export function authFormErrors(error: unknown): AuthFormErrors {
  if (isApiError(error) && usernameCodes.has(error.code)) {
    return { username: errorMessage(error) };
  }
  if (isApiError(error) && passwordCodes.has(error.code)) {
    return { password: errorMessage(error) };
  }
  return { form: errorMessage(error) };
}
