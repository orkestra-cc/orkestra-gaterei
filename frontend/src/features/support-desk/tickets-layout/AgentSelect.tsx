import { Form } from 'react-bootstrap';

export const AgentSelect = ({ agent, className = '' }) => {
  return (
    <Form.Select
      style={{ width: '9.375rem' }}
      className={className}
      size="sm"
      defaultValue={agent}
    >
      {['Select Agent', 'Anindya', 'Nowrin', 'Khalid'].map(item => (
        <option key={item}>{item}</option>
      ))}
    </Form.Select>
  );
};

export default AgentSelect;
