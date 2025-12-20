
import { useGetUserByIdQuery } from 'store/api/userApi';
import { useSelector } from 'react-redux';
import { selectUser } from 'store/slices/authSlice';
import OperatoreBanner from './OperatoreBanner';
import OperatoreProfileIntro from './OperatoreProfileIntro';
import { Col, Row, Alert, Spinner } from 'react-bootstrap';
import OperatoreActivityLog from './OperatoreActivityLog';

const OperatoreProfile: React.FC = () => {
  const currentUser = useSelector(selectUser);
  const userId = currentUser?.id;

  const {
    data: user,
    isLoading,
    error
  } = useGetUserByIdQuery(userId!, {
    skip: !userId
  });

  if (isLoading) {
    return (
      <div className="d-flex justify-content-center align-items-center" style={{ minHeight: '400px' }}>
        <Spinner animation="border" role="status">
          <span className="visually-hidden">Caricamento...</span>
        </Spinner>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="danger">
        Errore nel caricamento dei dati utente. Riprova più tardi.
      </Alert>
    );
  }

  if (!user) {
    return (
      <Alert variant="warning">
        Utente non trovato.
      </Alert>
    );
  }

  return (
    <>
      <OperatoreBanner user={user} />
      <Row className="g-3 mb-3">
        <Col lg={12}>
          <OperatoreProfileIntro user={user} />
          <OperatoreActivityLog className="mt-3" userId={userId!} />
        </Col>
      </Row>
    </>
  );
};

export default OperatoreProfile;