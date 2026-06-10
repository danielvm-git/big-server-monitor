import type { processmonitor } from "../../wailsjs/go/models";
import ProjectsSection from "./ProjectsSection";
import ServerRow from "./ServerRow";

interface ServerListProps {
  servers: processmonitor.Server[];
  onKill: (pid: number) => void;
}

function groupByProject(servers: processmonitor.Server[]) {
  const withProject = new Map<
    string,
    { dir: string; servers: processmonitor.Server[] }
  >();
  const withoutProject: processmonitor.Server[] = [];

  for (const server of servers) {
    if (server.projectName) {
      const existing = withProject.get(server.projectName);
      if (existing) {
        existing.servers.push(server);
      } else {
        withProject.set(server.projectName, {
          dir: server.projectDir,
          servers: [server],
        });
      }
    } else {
      withoutProject.push(server);
    }
  }

  return { withProject, withoutProject };
}

export default function ServerList({ servers, onKill }: ServerListProps) {
  if (servers.length === 0) {
    return (
      <div className="pk-server-list">
        <div className="pk-server-list-empty">No servers detected</div>
      </div>
    );
  }

  const { withProject, withoutProject } = groupByProject(servers);

  return (
    <div className="pk-server-list">
      {Array.from(withProject.entries()).map(
        ([projectName, { dir, servers: projectServers }]) => (
          <ProjectsSection
            key={projectName}
            projectName={projectName}
            projectDir={dir}
            servers={projectServers}
            onKill={onKill}
          />
        ),
      )}
      {withoutProject.map((server) => (
        <ServerRow key={server.port} server={server} onKill={onKill} />
      ))}
    </div>
  );
}
