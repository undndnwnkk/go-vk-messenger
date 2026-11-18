import { useMutation } from "@tanstack/react-query";
import { useState, type SubmitEvent } from "react";

import type { Credentials } from "../../api/types";
import { useAuth } from "../../auth/useAuth";
import { Button } from "../../components/Button";
import { TextField } from "../../components/TextField";
import { AuthLayout } from "./AuthLayout";
import { authFormErrors } from "./authErrors";
import { FormAlert } from "./FormAlert";
import { PasswordField } from "./PasswordField";

export function RegisterPage() {
  const { register } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const mutation = useMutation({
    mutationFn: (credentials: Credentials) => register(credentials, true),
  });
  const errors = mutation.isError ? authFormErrors(mutation.error) : {};

  const onSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate({ username: username.trim().toLowerCase(), password });
  };

  return (
    <AuthLayout
      title="Create your account"
      subtitle="Pick a username so people can find you."
      footer={null}
    >
      <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
        {errors.form && <FormAlert message={errors.form} />}
        <TextField
          label="Username"
          name="username"
          prefix="@"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          required
          value={username}
          error={errors.username}
          hint="3 to 32 lowercase letters, digits or underscores."
          onChange={(event) => {
            setUsername(event.target.value);
          }}
        />
        <PasswordField
          label="Password"
          name="password"
          autoComplete="new-password"
          required
          value={password}
          error={errors.password}
          hint="Use 8 to 72 characters."
          onChange={(event) => {
            setPassword(event.target.value);
          }}
        />
        <Button
          type="submit"
          size="lg"
          disabled={mutation.isPending || username.trim() === "" || password === ""}
        >
          {mutation.isPending ? "Creating account…" : "Create account"}
        </Button>
      </form>
    </AuthLayout>
  );
}
