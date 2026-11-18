import { useMutation } from "@tanstack/react-query";
import { useState, type SubmitEvent } from "react";
import { Link } from "react-router";

import type { Credentials } from "../../api/types";
import { useAuth } from "../../auth/useAuth";
import { Button } from "../../components/Button";
import { TextField } from "../../components/TextField";
import { AuthLayout } from "./AuthLayout";
import { authFormErrors } from "./authErrors";
import { FormAlert } from "./FormAlert";
import { PasswordField } from "./PasswordField";

export function LoginPage() {
  const { login } = useAuth();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [remember, setRemember] = useState(true);
  const mutation = useMutation({
    mutationFn: (credentials: Credentials) => login(credentials, remember),
  });
  const errors = mutation.isError ? authFormErrors(mutation.error) : {};

  const onSubmit = (event: SubmitEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate({ username: username.trim(), password });
  };

  return (
    <AuthLayout
      title="Welcome back"
      subtitle="Sign in to continue to your conversations."
      footer={
        <p className="m-0 text-base text-text-2">
          New to Plover?{" "}
          <Link to="/register" className="font-semibold text-accent no-underline">
            Create an account
          </Link>
        </p>
      }
    >
      <form className="flex flex-col gap-4" onSubmit={onSubmit} noValidate>
        {errors.form && <FormAlert message={errors.form} />}
        <TextField
          label="Username"
          name="username"
          autoComplete="username"
          autoCapitalize="none"
          spellCheck={false}
          required
          value={username}
          error={errors.username}
          onChange={(event) => {
            setUsername(event.target.value);
          }}
        />
        <PasswordField
          label="Password"
          name="password"
          autoComplete="current-password"
          required
          value={password}
          error={errors.password}
          onChange={(event) => {
            setPassword(event.target.value);
          }}
        />
        <label className="flex items-center gap-2.5 text-base text-text-2">
          <input
            type="checkbox"
            checked={remember}
            onChange={(event) => {
              setRemember(event.target.checked);
            }}
            className="m-0 size-5 flex-none accent-accent"
          />
          Keep me signed in
        </label>
        <Button
          type="submit"
          size="lg"
          disabled={mutation.isPending || username.trim() === "" || password === ""}
        >
          {mutation.isPending ? "Signing in…" : "Sign in"}
        </Button>
      </form>
    </AuthLayout>
  );
}
