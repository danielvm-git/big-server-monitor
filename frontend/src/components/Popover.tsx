import type { ReactNode } from "react";
import "../styles/popover.css";

interface PopoverProps {
  children: ReactNode;
}

export default function Popover({ children }: PopoverProps) {
  return <div className="pk-popover">{children}</div>;
}
