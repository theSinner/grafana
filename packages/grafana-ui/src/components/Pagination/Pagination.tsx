import React from 'react';
import { css } from 'emotion';
import { stylesFactory } from '../../themes';
import { Button, ButtonVariant } from '../Button';
import { Icon } from '../Icon/Icon';

const PAGE_LENGTH_TO_CONDENSE = 10;

interface Props {
  /** The current page index being shown.  */
  currentPage: number;
  /** Number of total pages.  */
  numberOfPages: number;
  /** Callback function for fetching the selected page  */
  onNavigate: (toPage: number) => void;
}

export const Pagination: React.FC<Props> = ({ currentPage, numberOfPages, onNavigate }) => {
  const styles = getStyles();
  const pages = [...new Array(numberOfPages).keys()];

  const condensePages = numberOfPages >= PAGE_LENGTH_TO_CONDENSE;
  const getListItem = (page: number, variant: 'primary' | 'secondary') => (
    <li key={page} className={styles.item}>
      <Button size="sm" variant={variant} onClick={() => onNavigate(page)}>
        {page}
      </Button>
    </li>
  );

  return (
    <div className={styles.container}>
      <ol>
        <li className={styles.item}>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => onNavigate(currentPage - 1)}
            disabled={currentPage === 1}
          >
            <Icon name="angle-left" />
          </Button>
        </li>
        {pages.reduce<JSX.Element[]>((pagesToRender, pageIndex) => {
          const page = pageIndex + 1;
          const variant: ButtonVariant = page === currentPage ? 'primary' : 'secondary';

          if (condensePages) {
            if (page === 1 || page === numberOfPages || (page >= currentPage - 2 && page <= currentPage + 2)) {
              pagesToRender.push(getListItem(page, variant));
            } else if (page === currentPage - 3 || page === currentPage + 3) {
              pagesToRender.push(
                <li key={page} className={styles.item}>
                  <Icon className={styles.ellipsis} name="ellipsis-v" />
                </li>
              );
            }
          } else {
            pagesToRender.push(getListItem(page, variant));
          }
          return pagesToRender;
        }, [])}
        <li className={styles.item}>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => onNavigate(currentPage + 1)}
            disabled={currentPage === numberOfPages}
          >
            <Icon name="angle-right" />
          </Button>
        </li>
      </ol>
    </div>
  );
};

const getStyles = stylesFactory(() => {
  return {
    container: css`
      float: right;
    `,
    item: css`
      display: inline-block;
      padding-left: 10px;
      margin-bottom: 5px;
    `,
    ellipsis: css`
      transform: rotate(90deg);
    `,
  };
});
