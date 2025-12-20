import React, { useState } from 'react';
import { Col, Form, Row } from 'react-bootstrap';
import DatePicker from 'react-datepicker';

// Types for Input Field components
interface InputFieldProps {
  label: string;
  type?: string;
  name: string;
  handleChange?: (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => void;
  onChange?: (value: string) => void;
  value?: string;
  as?: string;
  rows?: number;
  [key: string]: any; // For rest props
}

const DateInputField: React.FC = () => {
  const [date, setDate] = useState<Date | null>(null);

  return (
    <DatePicker
      selected={date}
      onChange={(date: Date | null) => setDate(date)}
      className="form-control form-control-sm"
      placeholderText="Select Date"
    />
  );
};

const InputField: React.FC<InputFieldProps> = ({ label, type = 'text', name, handleChange, ...rest }) => (
  <Form.Group as={Row} className="mb-3" controlId={name}>
    <Form.Label column sm={3} className="text-lg-end">
      {label}
    </Form.Label>
    <Col sm={9} md={7}>
      {type === 'date' ? (
        <DateInputField />
      ) : (
        <Form.Control
          type={type}
          placeholder={label}
          size="sm"
          name={name}
          onChange={handleChange}
          {...rest}
        />
      )}
    </Col>
  </Form.Group>
);

export default InputField;
