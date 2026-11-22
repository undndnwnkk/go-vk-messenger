import { useState, type InputHTMLAttributes } from "react";

import { IconButton } from "../../components/IconButton";
import { TextField } from "../../components/TextField";

interface PasswordFieldProps extends Omit<
  InputHTMLAttributes<HTMLInputElement>,
  "id" | "type" | "prefix"
> {
  label: string;
  error?: string | undefined;
  hint?: string | undefined;
}

export function PasswordField(props: PasswordFieldProps) {
  const [visible, setVisible] = useState(false);
  return (
    <TextField
      {...props}
      type={visible ? "text" : "password"}
      trailing={
        <IconButton
          label={visible ? "Hide password" : "Show password"}
          icon={visible ? "eyeOff" : "eye"}
          onClick={() => {
            setVisible((value) => !value);
          }}
          className="rounded-[10px]"
        />
      }
    />
  );
}
