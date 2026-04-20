package api

const MutationIssueUpdate = `
mutation IssueUpdate($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
    issue {
      id identifier title description url priority
      state { id name type }
      assignee { id name }
      team { id name key }
      createdAt updatedAt archivedAt
    }
  }
}`
