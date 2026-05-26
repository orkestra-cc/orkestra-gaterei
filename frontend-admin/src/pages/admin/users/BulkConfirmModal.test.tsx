import { describe, it, expect, vi, beforeEach } from 'vitest';
import { http, HttpResponse } from 'msw';
import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from 'test/render';
import { server } from 'test/server';
import { url } from 'test/handlers';
import UserTable from 'pages/admin/users/UserTable';
import type { User, UserListResponse } from 'store/api/userApi';

// Fixture: an administrator who's logged in (self) + two other operators
// the bulk action will target. We don't include the admin among the
// selected rows because the test scenarios that need the self-exclusion
// path will explicitly select the admin row.
const adminSelf: User = {
  id: 'admin-1',
  email: 'admin@example.com',
  username: 'admin',
  fullName: 'Admin Self',
  role: 'administrator',
  providers: [],
  isActive: true,
  emailVerified: true,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z'
};

const opOne: User = {
  id: 'op-1',
  email: 'op1@example.com',
  username: 'op1',
  fullName: 'Op One',
  role: 'operator',
  providers: [],
  isActive: true,
  emailVerified: true,
  createdAt: '2026-01-02T00:00:00Z',
  updatedAt: '2026-01-02T00:00:00Z'
};

const opTwo: User = {
  id: 'op-2',
  email: 'op2@example.com',
  username: 'op2',
  fullName: 'Op Two',
  role: 'operator',
  providers: [],
  isActive: true,
  emailVerified: true,
  createdAt: '2026-01-03T00:00:00Z',
  updatedAt: '2026-01-03T00:00:00Z'
};

const listResponse = (users: User[]): UserListResponse => ({
  users,
  total: users.length,
  page: 1,
  pageSize: 10,
  totalPages: 1
});

const listHandler = (users: User[]) =>
  http.get(url('/v1/users'), () => HttpResponse.json(listResponse(users)));

const preloadedAuthState = {
  auth: {
    user: {
      id: adminSelf.id,
      email: adminSelf.email,
      username: adminSelf.username,
      fullName: adminSelf.fullName,
      role: adminSelf.role,
      providers: [],
      isActive: true,
      emailVerified: true,
      createdAt: adminSelf.createdAt,
      updatedAt: adminSelf.updatedAt
    },
    isAuthenticated: true,
    isLoading: false,
    error: null,
    sessionExpiry: null,
    permissions: [] as string[],
    preferences: {
      theme: 'light' as const,
      language: 'en',
      notifications: true
    },
    _isLoggingOut: false,
    accessToken: 'test-token',
    tokenExpiry: null
  }
};

// Click the data-row checkbox for the row containing fullName. The header
// row has a master-select checkbox, which is what we'd hit if we matched
// by accessor alone — so we scope to the matching <tr>.
async function selectRow(
  user: ReturnType<typeof userEvent.setup>,
  fullName: string
) {
  const row = screen.getByText(fullName).closest('tr');
  if (!row) throw new Error(`row for ${fullName} not found`);
  // Bootstrap Form.Check.Input renders a real <input type=checkbox>.
  const checkbox = within(row as HTMLElement).getByRole('checkbox');
  await user.click(checkbox);
}

describe('Bulk actions flow', () => {
  beforeEach(() => {
    server.use(listHandler([adminSelf, opOne, opTwo]));
  });

  it('Apply is disabled until a bulk action is picked', async () => {
    renderWithProviders(<UserTable />, { preloadedState: preloadedAuthState });
    await waitFor(() => expect(screen.getByText('Op One')).toBeInTheDocument());

    const user = userEvent.setup();
    await selectRow(user, 'Op One');

    const apply = screen.getByRole('button', { name: /apply/i });
    expect(apply).toBeDisabled();
  });

  it('bulk-delete fans out one DELETE per selected non-self user', async () => {
    const deleted = vi.fn();
    server.use(
      http.delete(url('/v1/users/:id'), ({ params }) => {
        deleted(params.id);
        return HttpResponse.json({ success: true, message: 'gone' });
      })
    );

    renderWithProviders(<UserTable />, { preloadedState: preloadedAuthState });
    await waitFor(() => expect(screen.getByText('Op One')).toBeInTheDocument());

    const user = userEvent.setup();
    await selectRow(user, 'Op One');
    await selectRow(user, 'Op Two');

    // Pick "Delete" from the bulk-action select.
    const select = screen.getByRole('combobox', { name: /bulk actions/i });
    await user.selectOptions(select, 'delete');
    await user.click(screen.getByRole('button', { name: /apply/i }));

    // Confirm modal mounts. Find the dialog by the typed-confirm input.
    const confirmInput = await screen.findByPlaceholderText('Type DELETE');
    const dialog = confirmInput.closest('.modal') as HTMLElement;
    expect(dialog).toBeTruthy();

    // Run button is disabled until DELETE is typed.
    const run = within(dialog).getByRole('button', { name: /^delete$/i });
    expect(run).toBeDisabled();
    await user.type(confirmInput, 'DELETE');
    expect(run).toBeEnabled();

    await user.click(run);

    await waitFor(() => expect(deleted).toHaveBeenCalledTimes(2));
    expect(deleted).toHaveBeenCalledWith('op-1');
    expect(deleted).toHaveBeenCalledWith('op-2');
    // The self row was never selected here; the assertion above checking
    // exactly 2 calls covers that. Selecting the admin row is exercised
    // by the next test.
  });

  it('excludes the current user from a bulk delete and shows the notice', async () => {
    const deleted = vi.fn();
    server.use(
      http.delete(url('/v1/users/:id'), ({ params }) => {
        deleted(params.id);
        return HttpResponse.json({ success: true, message: 'gone' });
      })
    );

    renderWithProviders(<UserTable />, { preloadedState: preloadedAuthState });
    await waitFor(() => expect(screen.getByText('Op One')).toBeInTheDocument());

    const user = userEvent.setup();
    await selectRow(user, 'Admin Self');
    await selectRow(user, 'Op One');

    await user.selectOptions(
      screen.getByRole('combobox', { name: /bulk actions/i }),
      'delete'
    );
    await user.click(screen.getByRole('button', { name: /apply/i }));

    const confirmInput = await screen.findByPlaceholderText('Type DELETE');
    const dialog = confirmInput.closest('.modal') as HTMLElement;

    // Self-skip notice appears in the modal.
    expect(
      within(dialog).getByText(
        /Your own account.*admin@example\.com.*excluded/i
      )
    ).toBeInTheDocument();

    await user.type(confirmInput, 'DELETE');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    await waitFor(() => expect(deleted).toHaveBeenCalledTimes(1));
    expect(deleted).toHaveBeenCalledWith('op-1');
    expect(deleted).not.toHaveBeenCalledWith('admin-1');
  });

  it('summarizes partial failures in the modal', async () => {
    server.use(
      http.delete(url('/v1/users/op-1'), () =>
        HttpResponse.json({ success: true, message: 'gone' })
      ),
      http.delete(url('/v1/users/op-2'), () =>
        HttpResponse.json(
          {
            status: 403,
            title: 'Forbidden',
            detail: 'Refusing to remove the last active administrator',
            code: 'user.last_admin_forbidden'
          },
          { status: 403 }
        )
      )
    );

    renderWithProviders(<UserTable />, { preloadedState: preloadedAuthState });
    await waitFor(() => expect(screen.getByText('Op One')).toBeInTheDocument());

    const user = userEvent.setup();
    await selectRow(user, 'Op One');
    await selectRow(user, 'Op Two');

    await user.selectOptions(
      screen.getByRole('combobox', { name: /bulk actions/i }),
      'delete'
    );
    await user.click(screen.getByRole('button', { name: /apply/i }));

    const confirmInput = await screen.findByPlaceholderText('Type DELETE');
    const dialog = confirmInput.closest('.modal') as HTMLElement;
    await user.type(confirmInput, 'DELETE');
    await user.click(within(dialog).getByRole('button', { name: /^delete$/i }));

    // Failure list shows up with the translated last-admin code message
    // (the code is the only string unique to the failure pane — Op Two's
    // name also appears in the unaffected preview list above it).
    expect(
      await within(dialog).findByText(/last active administrator/i)
    ).toBeInTheDocument();

    // The Cancel/Close button label flips to "Close" after completion.
    // Scope to the footer so we don't match the modal's own X (aria
    // "Close") button at the top right.
    const footer = dialog.querySelector('.modal-footer') as HTMLElement;
    expect(
      within(footer).getByRole('button', { name: /close/i })
    ).toBeInTheDocument();
  });
});
