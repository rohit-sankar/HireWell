import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TextField } from "@/components/ui/form-field";
import { Alert } from "@/components/ui/alert";

describe("TextField", () => {
  it("renders a label associated with its input", () => {
    render(<TextField label="Email" value="" onChange={() => {}} />);
    const input = screen.getByLabelText("Email");
    expect(input).toBeInTheDocument();
  });

  it("displays a validation error message when provided", () => {
    render(<TextField label="Password" value="" onChange={() => {}} error="Password is required" />);
    expect(screen.getByText("Password is required")).toBeInTheDocument();
  });

  it("marks the input as aria-invalid when an error is present", () => {
    render(<TextField label="Email" value="" onChange={() => {}} error="Invalid email" />);
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "true");
  });

  it("does not mark aria-invalid when there is no error", () => {
    render(<TextField label="Email" value="" onChange={() => {}} />);
    expect(screen.getByLabelText("Email")).toHaveAttribute("aria-invalid", "false");
  });

  it("calls onChange when the user types", async () => {
    const user = userEvent.setup();
    let value = "";
    const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
      value = e.target.value;
    };
    render(<TextField label="Full name" value={value} onChange={onChange} />);
    await user.type(screen.getByLabelText("Full name"), "J");
    expect(value).toBe("J");
  });
});

describe("Alert", () => {
  it("renders its children", () => {
    render(<Alert>Something went wrong</Alert>);
    expect(screen.getByText("Something went wrong")).toBeInTheDocument();
  });

  it("defaults to error styling", () => {
    render(<Alert>Error message</Alert>);
    expect(screen.getByText("Error message").className).toContain("red");
  });

  it("applies success styling when tone='success'", () => {
    render(<Alert tone="success">Saved successfully</Alert>);
    expect(screen.getByText("Saved successfully").className).toContain("emerald");
  });
});
