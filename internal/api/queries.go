package api

// GraphQL query constants. Field shape matches types decoded in sync package.

const QueryViewerAndTeams = `
query ViewerAndTeams {
  viewer { id name email }
  teams(first: 50) {
    nodes { id name key description }
  }
}`

const QueryWorkspaceUsers = `
query WorkspaceUsers($first: Int!, $after: String) {
  users(first: $first, after: $after) {
    pageInfo { hasNextPage endCursor }
    nodes { id name email isMe active }
  }
}`

const QueryTeamWorkflowStates = `
query TeamWorkflowStates($teamId: String!) {
  team(id: $teamId) {
    id
    states(first: 100) {
      nodes { id name type color }
    }
  }
}`

const QueryMyIssues = `
query MyIssues($first: Int!, $after: String) {
  viewer {
    id
    assignedIssues(
      first: $first
      after: $after
      filter: { state: { type: { nin: ["completed", "canceled"] } } }
    ) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id identifier title description url priority
        state { id name type }
        assignee { id name }
        team { id name key }
        createdAt updatedAt archivedAt
      }
    }
  }
}`

const QueryTeamTriage = `
query TeamTriage($teamId: String!, $first: Int!, $after: String) {
  team(id: $teamId) {
    id
    issues(
      first: $first
      after: $after
      filter: {
        assignee: { null: true }
        state: { type: { in: ["triage", "backlog"] } }
      }
    ) {
      pageInfo { hasNextPage endCursor }
      nodes {
        id identifier title description url priority
        state { id name type }
        team { id name key }
        createdAt updatedAt archivedAt
      }
    }
  }
}`

const QueryIssueDetail = `
query IssueDetail($id: String!) {
  issue(id: $id) {
    id identifier title description url priority
    state { id name type }
    assignee { id name }
    team { id name key }
    createdAt updatedAt archivedAt
  }
}`
