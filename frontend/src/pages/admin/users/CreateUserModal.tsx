import React, { useState } from 'react';
import { Modal, Button, Form, Alert } from 'react-bootstrap';
import { useCreateUserMutation, CreateUserInput } from 'store/api/userApi';
import FalconCloseButton from 'components/common/FalconCloseButton';

interface CreateUserModalProps {
  show: boolean;
  onHide: () => void;
  onSuccess?: () => void;
}

const CreateUserModal: React.FC<CreateUserModalProps> = ({
  show,
  onHide,
  onSuccess
}) => {
  const [createUser, { isLoading }] = useCreateUserMutation();
  const [error, setError] = useState<string>('');
  const [formData, setFormData] = useState<CreateUserInput>({
    fullName: '',
    email: '',
    username: '',
    phone: '',
    pin: '',
    role: 'operator'
  });

  const roles = [
    { value: 'ceo', label: 'CEO' },
    { value: 'developer', label: 'Sviluppatore' },
    { value: 'administrator', label: 'Amministratore' },
    { value: 'manager', label: 'Manager' },
    { value: 'operator', label: 'Operatore' },
    { value: 'guest', label: 'Ospite' }
  ];

  const handleChange = (
    e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>
  ) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
    setError('');
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    // Validation
    if (!formData.fullName.trim()) {
      setError('Il nome completo è obbligatorio');
      return;
    }
    if (!formData.email.trim()) {
      setError("L'email è obbligatoria");
      return;
    }

    if (!formData.phone.trim()) {
      setError('Il telefono è obbligatorio');
      return;
    }
    // if (!formData.username.trim()) {
    //   setError("L'username è obbligatorio");
    //   return;
    // }
    // if (!formData.pin.trim()) {
    //   setError('Il PIN è obbligatorio');
    //   return;
    // }
    // if (!/^\d{5}$/.test(formData.pin)) {
    //   setError('Il PIN deve essere un numero di 5 cifre');
    //   return;
    // }

    try {
      await createUser(formData).unwrap();
      // Reset form
      setFormData({
        fullName: '',
        email: '',
        username: '',
        phone: '',
        pin: '',
        role: 'operator'
      });
      onHide();
      if (onSuccess) onSuccess();
    } catch (err: any) {
      setError(err?.data?.message || "Errore durante la creazione dell'utente");
    }
  };

  const handleClose = () => {
    setFormData({
      fullName: '',
      email: '',
      username: '',
      phone: '',
      pin: '',
      role: 'operator'
    });
    setError('');
    onHide();
  };

  return (
    <Modal show={show} onHide={handleClose} centered size="lg">
      <Modal.Header>
        <Modal.Title>Nuovo Utente</Modal.Title>
        <FalconCloseButton onClick={handleClose} />
      </Modal.Header>
      <Form onSubmit={handleSubmit}>
        <Modal.Body>
          {error && (
            <Alert variant="danger" dismissible onClose={() => setError('')}>
              {error}
            </Alert>
          )}

          <Form.Group className="mb-3">
            <Form.Label>
              Nome Completo <span className="text-danger">*</span>
            </Form.Label>
            <Form.Control
              type="text"
              name="fullName"
              value={formData.fullName}
              onChange={handleChange}
              placeholder="Es: Mario Rossi"
              required
            />
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>
              Email <span className="text-danger">*</span>
            </Form.Label>
            <Form.Control
              type="email"
              name="email"
              value={formData.email}
              onChange={handleChange}
              placeholder="mario.rossi@example.com"
              required
            />
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>
              Telefono <span className="text-danger">*</span>
            </Form.Label>
            <Form.Control
              type="tel"
              name="phone"
              value={formData.phone}
              onChange={handleChange}
              placeholder="+39 333 1234567"
              required
            />
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>Username</Form.Label>
            <Form.Control
              type="text"
              name="username"
              value={formData.username}
              onChange={handleChange}
              placeholder="mariorossi"
            />
            <Form.Text className="text-muted">
              L'username verrà utilizzato per accedere al sistema
            </Form.Text>
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>PIN</Form.Label>
            <Form.Control
              type="text"
              name="pin"
              value={formData.pin}
              onChange={handleChange}
              placeholder="12345"
              pattern="\d{5}"
              maxLength={5}
            />
            <Form.Text className="text-muted">
              Codice numerico di 5 cifre per l'accesso
            </Form.Text>
          </Form.Group>

          <Form.Group className="mb-3">
            <Form.Label>
              Ruolo <span className="text-danger">*</span>
            </Form.Label>
            <Form.Select
              name="role"
              value={formData.role}
              onChange={handleChange}
              required
            >
              {roles.map(role => (
                <option key={role.value} value={role.value}>
                  {role.label}
                </option>
              ))}
            </Form.Select>
          </Form.Group>
        </Modal.Body>
        <Modal.Footer>
          <Button
            variant="secondary"
            onClick={handleClose}
            disabled={isLoading}
          >
            Annulla
          </Button>
          <Button variant="primary" type="submit" disabled={isLoading}>
            {isLoading ? 'Creazione in corso...' : 'Crea Utente'}
          </Button>
        </Modal.Footer>
      </Form>
    </Modal>
  );
};

export default CreateUserModal;
