
import classNames from 'classnames';
import { Button, ButtonProps } from 'react-bootstrap';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { IconProp, Transform } from '@fortawesome/fontawesome-svg-core';

type IconAlign = 'left' | 'right' | 'middle';

interface IconButtonProps extends ButtonProps {
  icon: IconProp;
  iconAlign?: IconAlign;
  iconClassName?: string;
  transform?: string | Transform;
  children?: React.ReactNode;
  ref?: React.Ref<HTMLButtonElement>;
}

const IconButton: React.FC<IconButtonProps> = ({
  icon,
  iconAlign = 'left',
  iconClassName,
  transform,
  children,
  ref,
  ...rest
}) => (
  <Button {...rest} ref={ref}>
    {iconAlign === 'right' && children}
    <FontAwesomeIcon
      icon={icon}
      className={classNames(iconClassName, {
        'me-1': children && iconAlign === 'left',
        'ms-1': children && iconAlign === 'right'
      })}
      transform={transform}
    />
    {iconAlign === 'left' || iconAlign === 'middle' ? children : false}
  </Button>
);

export default IconButton;
