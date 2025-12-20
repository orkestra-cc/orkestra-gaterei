import { Card } from 'react-bootstrap';
import SocialLoginForm from 'components/authentication/SocialLoginForm';
import AuthCardLayout from 'layouts/AuthCardLayout';

const Login = () => (
  <AuthCardLayout>
    <Card>
      <Card.Body className="p-4 p-sm-5">
        <div className="text-center mb-4">
          <h3 className="mb-3">Benvenuto</h3>
          <p className="text-muted">
            Esegui l'accesso con il tuo account per continuare.
          </p>
        </div>
        <SocialLoginForm />
      </Card.Body>
    </Card>
  </AuthCardLayout>
);

export default Login;
