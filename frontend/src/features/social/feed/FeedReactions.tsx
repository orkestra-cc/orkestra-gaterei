import { Link } from 'react-router';

const LCSText = ({ count, text }) => (
  <Link className="text-700 me-1" to="#!">
    {count} {text}
    {count !== 1 && 's'}
  </Link>
);

const FeedReactions = ({ like, comment, share }) => {
  return (
    <div className="border-bottom border-200 fs-10 py-3">
      {!!like && <LCSText count={like} text="like" />}
      {!!comment && (
        <>
          • <LCSText count={comment} text="comment" />{' '}
        </>
      )}
      {!!share && (
        <>
          • <LCSText count={share} text="share" />{' '}
        </>
      )}
    </div>
  );
};

export default FeedReactions;
