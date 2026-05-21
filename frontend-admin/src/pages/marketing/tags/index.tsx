// Tags admin — flat list with create/edit/delete. Phase 1 ships the
// minimal slug-stable CRUD surface; tree visualisation and
// move-subtree arrive in a follow-up alongside the cascade on
// tags[] arrays of Person/Organization records.

import { useState } from 'react';
import {
  Button,
  Card,
  Form,
  Modal,
  Table,
  Badge,
  InputGroup
} from 'react-bootstrap';
import { Trans, useTranslation } from 'react-i18next';
import {
  useListMarketingTagsQuery,
  useCreateMarketingTagMutation,
  useUpdateMarketingTagMutation,
  useDeleteMarketingTagMutation
} from 'store/api/marketingApi';
import type { Tag, TagPayload } from 'types/marketing';

const empty: TagPayload = {
  name: '',
  slug: '',
  description: '',
  color: '',
  parentUuid: ''
};

const TagsPage: React.FC = () => {
  const { t } = useTranslation();
  const { data, isLoading } = useListMarketingTagsQuery();
  const [createTag] = useCreateMarketingTagMutation();
  const [updateTag] = useUpdateMarketingTagMutation();
  const [deleteTag] = useDeleteMarketingTagMutation();

  const [show, setShow] = useState(false);
  const [editing, setEditing] = useState<Tag | null>(null);
  const [form, setForm] = useState<TagPayload>(empty);

  const openNew = () => {
    setEditing(null);
    setForm(empty);
    setShow(true);
  };
  const openEdit = (tag: Tag) => {
    setEditing(tag);
    setForm({
      name: tag.name,
      slug: tag.slug,
      description: tag.description ?? '',
      color: tag.color ?? '',
      parentUuid: tag.parentUuid ?? ''
    });
    setShow(true);
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const payload: TagPayload = {
      ...form,
      parentUuid: form.parentUuid || undefined,
      slug: form.slug || undefined
    };
    if (editing) {
      await updateTag({
        id: editing.uuid,
        patch: payload as unknown as Record<string, unknown>
      });
    } else {
      await createTag(payload);
    }
    setShow(false);
  };

  const onDelete = async (tag: Tag) => {
    if (
      !window.confirm(t('marketing.tags.confirmDelete', { name: tag.name }))
    ) {
      return;
    }
    await deleteTag(tag.uuid);
  };

  const parents = (data?.items ?? []).filter(tag => !tag.parentUuid);

  return (
    <>
      <div className="mb-3 d-flex justify-content-between align-items-center">
        <div>
          <h3 className="fw-normal mb-1">{t('marketing.tags.title')}</h3>
          <p className="fs-10 text-muted mb-0">
            {t('marketing.tags.subtitle')}
          </p>
        </div>
        <Button variant="primary" onClick={openNew}>
          {t('marketing.tags.newTag')}
        </Button>
      </div>

      <Card>
        <Card.Body className="p-0">
          {isLoading ? (
            <div className="p-3 text-muted">{t('marketing.tags.loading')}</div>
          ) : !data?.items?.length ? (
            <div className="p-3 text-muted">
              <Trans
                i18nKey="marketing.tags.empty"
                components={{ code: <code /> }}
              />
            </div>
          ) : (
            <Table responsive hover className="mb-0">
              <thead className="bg-200">
                <tr>
                  <th>{t('marketing.tags.colName')}</th>
                  <th>{t('marketing.tags.colSlug')}</th>
                  <th>{t('marketing.tags.colPath')}</th>
                  <th>{t('marketing.tags.colColor')}</th>
                  <th style={{ width: 140 }} />
                </tr>
              </thead>
              <tbody>
                {data.items.map(tag => (
                  <tr key={tag.uuid}>
                    <td className="fw-medium">{tag.name}</td>
                    <td>
                      <code className="fs-10">{tag.slug}</code>
                    </td>
                    <td>
                      <small className="text-muted">{tag.path}</small>
                    </td>
                    <td>
                      {tag.color ? (
                        <Badge
                          pill
                          style={{
                            backgroundColor: tag.color,
                            color: '#fff'
                          }}
                        >
                          {tag.color}
                        </Badge>
                      ) : (
                        '—'
                      )}
                    </td>
                    <td className="text-end">
                      <Button
                        size="sm"
                        variant="link"
                        onClick={() => openEdit(tag)}
                      >
                        {t('marketing.tags.edit')}
                      </Button>
                      <Button
                        size="sm"
                        variant="link"
                        className="text-danger"
                        onClick={() => onDelete(tag)}
                      >
                        {t('marketing.tags.delete')}
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </Table>
          )}
        </Card.Body>
      </Card>

      <Modal show={show} onHide={() => setShow(false)}>
        <Form onSubmit={onSubmit}>
          <Modal.Header closeButton>
            <Modal.Title>
              {editing
                ? t('marketing.tags.modalTitleEdit')
                : t('marketing.tags.modalTitleNew')}
            </Modal.Title>
          </Modal.Header>
          <Modal.Body>
            <Form.Group className="mb-3">
              <Form.Label>{t('marketing.tags.formName')}</Form.Label>
              <Form.Control
                required
                value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
              />
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>{t('marketing.tags.formSlug')}</Form.Label>
              <InputGroup>
                <Form.Control
                  value={form.slug}
                  onChange={e => setForm({ ...form, slug: e.target.value })}
                  placeholder={t('marketing.tags.formSlugPlaceholder')}
                />
              </InputGroup>
              <Form.Text className="text-muted">
                {t('marketing.tags.formSlugHelp')}
              </Form.Text>
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>{t('marketing.tags.formDescription')}</Form.Label>
              <Form.Control
                as="textarea"
                rows={2}
                value={form.description ?? ''}
                onChange={e =>
                  setForm({ ...form, description: e.target.value })
                }
              />
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>{t('marketing.tags.formColor')}</Form.Label>
              <Form.Control
                type="color"
                value={form.color || '#1f6feb'}
                onChange={e => setForm({ ...form, color: e.target.value })}
                style={{ width: 80, height: 40 }}
              />
            </Form.Group>
            <Form.Group className="mb-3">
              <Form.Label>{t('marketing.tags.formParent')}</Form.Label>
              <Form.Select
                value={form.parentUuid ?? ''}
                onChange={e => setForm({ ...form, parentUuid: e.target.value })}
                disabled={!!editing}
              >
                <option value="">{t('marketing.tags.formParentRoot')}</option>
                {parents.map(p => (
                  <option key={p.uuid} value={p.uuid}>
                    {p.name} ({p.slug})
                  </option>
                ))}
              </Form.Select>
              {editing && (
                <Form.Text className="text-muted">
                  {t('marketing.tags.formParentHelp')}
                </Form.Text>
              )}
            </Form.Group>
          </Modal.Body>
          <Modal.Footer>
            <Button variant="outline-secondary" onClick={() => setShow(false)}>
              {t('marketing.tags.cancel')}
            </Button>
            <Button type="submit" variant="primary">
              {editing ? t('marketing.tags.save') : t('marketing.tags.create')}
            </Button>
          </Modal.Footer>
        </Form>
      </Modal>
    </>
  );
};

export default TagsPage;
