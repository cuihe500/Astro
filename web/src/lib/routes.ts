export const projectsPath = "/projects";

function segment(value: string | number): string {
  return encodeURIComponent(String(value));
}

export function projectAppsPath(projectId: string | number): string {
  return `${projectsPath}/${segment(projectId)}/apps`;
}

export function createAppPath(projectId: string | number): string {
  return `${projectAppsPath(projectId)}/new`;
}

export function appDetailPath(projectId: string | number, appId: string | number): string {
  return `${projectAppsPath(projectId)}/${segment(appId)}`;
}
