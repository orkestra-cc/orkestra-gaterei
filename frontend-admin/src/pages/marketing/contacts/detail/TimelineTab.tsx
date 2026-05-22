// TimelineTab — Phase 2 contact-detail tab that renders the activity
// log for the person, with a "Log activity" button that opens the
// ManualActivityModal.
//
// Pagination: simple "Load more" button bumping skip — the timeline
// is bounded per-person and operators rarely scroll past the top 100
// rows. A virtualised infinite-scroll list is a follow-up if a real
// tenant hits the limit.

import { useState } from 'react';
import { Badge, Button, Table } from 'react-bootstrap';
import { useTranslation } from 'react-i18next';
import { useListPersonActivitiesQuery } from 'store/api/marketingApi';
import ManualActivityModal from './ManualActivityModal';

interface TimelineTabProps {
  personId: string;
}

const TimelineTab: React.FC<TimelineTabProps> = ({ personId }) => {
  const { t } = useTranslation();
  const [showModal, setShowModal] = useState(false);
  const [limit, setLimit] = useState(50);

  const { data, isLoading, error } = useListPersonActivitiesQuery({
    personId,
    limit
  });

  return (
    <>
      <div className="d-flex justify-content-between align-items-center mb-3">
        <p className="text-muted mb-0">{t('marketing.timeline.description')}</p>
        <Button
          size="sm"
          variant="outline-primary"
          onClick={() => setShowModal(true)}
        >
          {t('marketing.timeline.logActivity')}
        </Button>
      </div>

      {isLoading && (
        <p className="text-muted">{t('marketing.timeline.loading')}</p>
      )}
      {error && (
        <p className="text-danger">{t('marketing.timeline.loadError')}</p>
      )}

      {!isLoading && !error && (!data?.items || data.items.length === 0) && (
        <p className="text-muted mb-0">{t('marketing.timeline.empty')}</p>
      )}

      {data?.items && data.items.length > 0 && (
        <>
          <Table size="sm" responsive className="mb-2">
            <thead>
              <tr>
                <th style={{ width: 140 }}>
                  {t('marketing.timeline.colWhen')}
                </th>
                <th style={{ width: 140 }}>
                  {t('marketing.timeline.colKind')}
                </th>
                <th>{t('marketing.timeline.colPayload')}</th>
                <th style={{ width: 100 }}>
                  {t('marketing.timeline.colSource')}
                </th>
              </tr>
            </thead>
            <tbody>
              {data.items.map(a => (
                <tr key={a.uuid}>
                  <td>
                    <small className="text-muted">
                      {new Date(a.occurredAt).toLocaleString()}
                    </small>
                  </td>
                  <td>
                    <Badge
                      bg={a.kind === 'corrected_by' ? 'warning' : 'secondary'}
                      pill
                    >
                      {a.kind}
                    </Badge>
                  </td>
                  <td>
                    {a.payload && Object.keys(a.payload).length > 0 ? (
                      <small className="font-monospace">
                        {JSON.stringify(a.payload)}
                      </small>
                    ) : (
                      <small className="text-muted">—</small>
                    )}
                  </td>
                  <td>
                    <small className="text-muted">{a.source}</small>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>

          {data.meta.count >= limit && (
            <div className="d-flex justify-content-center">
              <Button
                size="sm"
                variant="link"
                onClick={() => setLimit(limit + 50)}
              >
                {t('marketing.timeline.loadMore')}
              </Button>
            </div>
          )}
        </>
      )}

      <ManualActivityModal
        personId={personId}
        show={showModal}
        onHide={() => setShowModal(false)}
      />
    </>
  );
};

export default TimelineTab;
