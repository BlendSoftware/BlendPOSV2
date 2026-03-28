/**
 * BrandMark — shared brand mark used in Login and Register pages.
 * Uses the real brand logo from /logo-mark.svg (cropped from /logo.svg).
 */
import classes from './BrandMark.module.css';

interface BrandMarkProps {
    /** 'row' = icon + text side by side, 'col' = stacked */
    layout?: 'row' | 'col';
    /** Icon height in px (default 52) */
    size?: number;
}

export function BrandMark({ layout = 'row', size = 52 }: BrandMarkProps) {
    return (
        <div className={`${classes.root} ${layout === 'col' ? classes.col : classes.row}`}>
            {/* Real brand mark from /logo-mark.svg */}
            <img
                src="/logo-mark.svg"
                alt="BlendPOS mark"
                className={classes.mark}
                style={{ height: `${size}px` }}
            />
            <div className={classes.wordmark}>
                <span className={classes.word}>Blend</span>
                <span className={classes.accent}>POS</span>
            </div>
        </div>
    );
}
