import { useEffect, useMemo, useState } from 'react';
import { Link } from 'react-router';
import { Card, Dropdown, Form } from 'react-bootstrap';
import SimpleBar from 'simplebar-react';
import { useAppDispatch, useAppSelector } from 'store/hooks';
import {
  selectCurrentMembership,
  selectImpersonation,
  selectIsImpersonating,
  selectMemberships,
  setCurrentOrg,
  startImpersonation,
  stopImpersonation
} from 'store/slices/tenantSlice';
import { useListAllOrgsAdminQuery } from 'store/api/tenantApi';
import { useAuth } from 'hooks/auth/useAuthRTK';
import { baseApi, TENANT_SCOPED_TAGS } from 'store/api/baseApi';

type TierFilter = 'all' | 'internal' | 'external';

/**
 * NineDotMenu hosts the workspace + operator-impersonation switchers that
 * used to sit as standalone dropdowns in the navbar. The dropdown is hidden
 * when the user has a single membership and cannot impersonate — there's
 * nothing to switch between in that case.
 *
 * Impersonation is gated server-side by system.tenants.admin; the UI gate
 * here is a convenience layer over that permission check.
 */
const NineDotMenu = () => {
  const dispatch = useAppDispatch();
  const { hasPermission } = useAuth();
  const memberships = useAppSelector(selectMemberships);
  const currentMembership = useAppSelector(selectCurrentMembership);
  const isImpersonating = useAppSelector(selectIsImpersonating);
  const impersonation = useAppSelector(selectImpersonation);

  const canImpersonate =
    hasPermission('system.tenants.admin') &&
    currentMembership?.kind === 'internal';

  const [show, setShow] = useState<boolean>(false);
  const [search, setSearch] = useState('');
  const [tierFilter, setTierFilter] = useState<TierFilter>('all');

  const { data, isLoading } = useListAllOrgsAdminQuery(undefined, {
    skip: !canImpersonate || !show
  });

  const filtered = useMemo(() => {
    const rows = data?.tenants ?? [];
    return rows.filter(t => {
      if (tierFilter !== 'all' && t.kind !== tierFilter) return false;
      if (!search) return true;
      const q = search.toLowerCase();
      return (
        t.name.toLowerCase().includes(q) ||
        t.slug.toLowerCase().includes(q) ||
        t.id.toLowerCase().includes(q)
      );
    });
  }, [data, search, tierFilter]);

  useEffect(() => {
    const onScroll = () => {
      if (window.innerWidth < 1200) setShow(false);
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    return () => window.removeEventListener('scroll', onScroll);
  }, []);

  if (memberships.length === 0) return null;
  if (memberships.length <= 1 && !canImpersonate) return null;

  const circleFill = isImpersonating
    ? 'var(--bs-warning)'
    : 'var(--bs-navbar-color, #6C6E71)';

  const onPickMembership = (tenantId: string) => {
    dispatch(setCurrentOrg(tenantId));
    setShow(false);
  };

  const onImpersonate = (tenantId: string, tenantName: string) => {
    dispatch(startImpersonation({ tenantId, tenantName }));
    dispatch(baseApi.util.invalidateTags([...TENANT_SCOPED_TAGS]));
    setShow(false);
  };

  const onStopImpersonation = () => {
    dispatch(stopImpersonation());
    dispatch(baseApi.util.invalidateTags([...TENANT_SCOPED_TAGS]));
  };

  return (
    <Dropdown
      navbar={true}
      as="li"
      show={show}
      onToggle={next => setShow(next)}
      autoClose="outside"
    >
      <Dropdown.Toggle
        bsPrefix="toggle"
        as={Link}
        to="#!"
        className="nav-link px-2 nine-dots"
        title={
          isImpersonating
            ? `Impersonating ${impersonation.tenantName ?? 'tenant'}`
            : 'Workspaces'
        }
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          width="16"
          height="37"
          viewBox="0 0 16 16"
          fill="none"
        >
          <circle cx="2" cy="2" r="2" fill={circleFill}></circle>
          <circle cx="2" cy="8" r="2" fill={circleFill}></circle>
          <circle cx="2" cy="14" r="2" fill={circleFill}></circle>
          <circle cx="8" cy="2" r="2" fill={circleFill}></circle>
          <circle cx="8" cy="8" r="2" fill={circleFill}></circle>
          <circle cx="8" cy="14" r="2" fill={circleFill}></circle>
          <circle cx="14" cy="2" r="2" fill={circleFill}></circle>
          <circle cx="14" cy="8" r="2" fill={circleFill}></circle>
          <circle cx="14" cy="14" r="2" fill={circleFill}></circle>
        </svg>
      </Dropdown.Toggle>

      <Dropdown.Menu
        className="dropdown-menu-card dropdown-menu-end dropdown-caret dropdown-caret-bg"
        show={show}
      >
        <Card className="dropdown-menu-end shadow-none" style={{ width: 340 }}>
          <SimpleBar style={{ maxHeight: 520 }}>
            <Card.Body className="p-0">
              <div className="px-3 pt-3 pb-2">
                <h6 className="mb-2 text-uppercase text-700 fs-11 fw-bold">
                  Your workspaces
                </h6>
                {memberships.map(m => {
                  const active = m.tenantId === currentMembership?.tenantId;
                  return (
                    <button
                      key={m.tenantId}
                      type="button"
                      className={`btn btn-sm w-100 text-start mb-1 ${
                        active
                          ? 'btn-soft-primary'
                          : 'btn-link text-decoration-none'
                      }`}
                      onClick={() => onPickMembership(m.tenantId)}
                    >
                      <div className="d-flex justify-content-between align-items-center">
                        <span className="fw-semibold text-truncate">
                          {m.name}
                        </span>
                        <small className="text-muted ms-2 flex-shrink-0">
                          {m.plan}
                        </small>
                      </div>
                      {m.roles.length > 0 && (
                        <small className="text-muted d-block text-truncate">
                          {m.roles.join(', ')}
                        </small>
                      )}
                    </button>
                  );
                })}
              </div>

              {canImpersonate && (
                <>
                  <hr className="my-0" />
                  <div className="px-3 pt-3 pb-3">
                    <h6 className="mb-2 text-uppercase text-700 fs-11 fw-bold">
                      Impersonate tenant
                    </h6>

                    {isImpersonating && (
                      <div className="alert alert-warning py-2 px-3 mb-2 d-flex justify-content-between align-items-center">
                        <small className="text-truncate">
                          Impersonating{' '}
                          <strong>{impersonation.tenantName}</strong>
                        </small>
                        <button
                          type="button"
                          className="btn btn-sm btn-warning ms-2 flex-shrink-0"
                          onClick={onStopImpersonation}
                        >
                          Stop
                        </button>
                      </div>
                    )}

                    <Form.Control
                      type="search"
                      size="sm"
                      placeholder="Search by name, slug, or id"
                      value={search}
                      onChange={e => setSearch(e.target.value)}
                    />
                    <div className="d-flex gap-1 mt-2 mb-2">
                      {(['all', 'internal', 'external'] as TierFilter[]).map(
                        t => (
                          <button
                            key={t}
                            type="button"
                            className={`btn btn-sm ${
                              tierFilter === t
                                ? 'btn-secondary'
                                : 'btn-outline-secondary'
                            }`}
                            onClick={() => setTierFilter(t)}
                          >
                            {t}
                          </button>
                        )
                      )}
                    </div>

                    {isLoading && (
                      <p className="text-muted small mb-0">Loading…</p>
                    )}
                    {!isLoading && filtered.length === 0 && (
                      <p className="text-muted small mb-0">No tenants match.</p>
                    )}
                    <div className="d-flex flex-column">
                      {filtered.slice(0, 200).map(t => {
                        const active = impersonation.tenantId === t.id;
                        return (
                          <button
                            key={t.id}
                            type="button"
                            className={`btn btn-sm text-start mb-1 ${
                              active
                                ? 'btn-soft-warning'
                                : 'btn-link text-decoration-none'
                            }`}
                            onClick={() => onImpersonate(t.id, t.name)}
                          >
                            <div className="d-flex justify-content-between align-items-center">
                              <span className="fw-semibold text-truncate">
                                {t.name}
                              </span>
                              <small className="text-muted ms-2 flex-shrink-0">
                                {t.kind ?? '—'}
                              </small>
                            </div>
                            <small className="text-muted d-block text-truncate">
                              {t.slug} · {t.plan}
                            </small>
                          </button>
                        );
                      })}
                    </div>
                    {filtered.length > 200 && (
                      <p className="text-muted small mb-0">
                        Showing first 200 of {filtered.length} — refine the
                        search.
                      </p>
                    )}
                  </div>
                </>
              )}
            </Card.Body>
          </SimpleBar>
        </Card>
      </Dropdown.Menu>
    </Dropdown>
  );
};

export default NineDotMenu;
