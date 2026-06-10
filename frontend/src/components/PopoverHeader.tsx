import { RotateCw } from "lucide-react";

interface PopoverHeaderProps {
  serverCount: number;
  onRefresh: () => void;
}

export default function PopoverHeader({
  serverCount,
  onRefresh,
}: PopoverHeaderProps) {
  return (
    <>
      <div className="pk-header">
        <div className="pk-header-left">
          <span className="pk-header-title">PortKeeper</span>
          <span className="pk-header-subtitle">
            {serverCount} {serverCount === 1 ? "server" : "servers"} active
          </span>
        </div>
        <button
          className="pk-header-refresh"
          onClick={onRefresh}
          aria-label="Refresh servers"
        >
          <RotateCw size={16} strokeWidth={2} />
        </button>
      </div>
      <div className="pk-header-divider" />
    </>
  );
}
