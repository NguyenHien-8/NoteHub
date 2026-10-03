#include "ImageGallery.h"
#include <QCache>
#include <QImageReader>
#include <QtConcurrent>
#include <QtWidgets>
#include <cmath>

namespace {
constexpr int gap = 10, footer = 30;
const char *mime = "application/x-notehub-attachment";
QCache<QString, QImage> cache(64 * 1024);
} // namespace

ImageGallery::ImageGallery(const QJsonArray &values, QWidget *parent) : QWidget(parent), attachments(values) {
    setAcceptDrops(true);
    setMouseTracking(true);
    QSizePolicy policy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    policy.setHeightForWidth(true);
    setSizePolicy(policy);
    for (const auto &value : values) {
        auto a = value.toObject();
        if (!a["mimeType"].toString().startsWith("image/"))
            continue;
        Tile tile{a["uid"].toString(), a["path"].toString(), a["filename"].toString(), {}};
        const auto key = tile.path + ":" + a["sha256"].toString();
        if (auto image = cache.object(key))
            tile.image = *image;
        const int index = tiles.size();
        tiles.append(tile);
        if (!tile.image.isNull() || tile.path.isEmpty())
            continue;
        auto watcher = new QFutureWatcher<QImage>(this);
        connect(watcher, &QFutureWatcher<QImage>::finished, this, [this, watcher, index, key] {
            const auto image = watcher->result();
            watcher->deleteLater();
            tiles[index].image = image;
            if (!image.isNull())
                cache.insert(key, new QImage(image), qMax(1, int(image.sizeInBytes() / 1024)));
            updateGeometry();
            update();
        });
        watcher->setFuture(QtConcurrent::run([path = tile.path] {
            QImageReader reader(path);
            reader.setAutoTransform(true);
            const auto size = reader.size();
            if (!size.isValid() || qint64(size.width()) * size.height() > 12000000)
                return QImage();
            reader.setScaledSize(size.scaled(800, 600, Qt::KeepAspectRatio));
            return reader.read();
        }));
    }
}
QVector<QRect> ImageGallery::frames(int width) const {
    QVector<QRect> result;
    if (tiles.isEmpty())
        return result;
    width = qMax(1, width);
    const int columns = qMin(int(tiles.size()), qMax(1, (width + gap) / 240));
    int y = 0;
    for (int first = 0; first < tiles.size(); first += columns) {
        const int end = qMin(int(tiles.size()), first + columns);
        qreal total = 0;
        QVector<qreal> ratios;
        for (int i = first; i < end; ++i) {
            const auto &image = tiles[i].image;
            qreal ratio = image.isNull() ? 1.4 : qreal(image.width()) / image.height();
            ratio = qBound(.65, ratio, 2.2);
            ratios.append(ratio);
            total += ratio;
        }
        const int available = width - (end - first - 1) * gap;
        const int height = qMin(300, int(available / total)) + footer + 12;
        int x = 0;
        for (int i = first; i < end; ++i) {
            int w = i == end - 1 ? width - x : int(available * ratios[i - first] / total);
            result.append({x, y, w, height});
            x += w + gap;
        }
        y += height + gap;
    }
    return result;
}
int ImageGallery::heightForWidth(int width) const {
    const auto r = frames(width);
    return r.isEmpty() ? 0 : r.last().bottom() + 1;
}
void ImageGallery::resizeEvent(QResizeEvent *event) {
    QWidget::resizeEvent(event);
    updateGeometry();
}
void ImageGallery::paintEvent(QPaintEvent *) {
    QPainter p(this);
    p.setRenderHint(QPainter::Antialiasing);
    p.setRenderHint(QPainter::SmoothPixmapTransform);
    const auto boxes = frames(width());
    for (int i = 0; i < tiles.size(); ++i) {
        const auto r = boxes[i].adjusted(1, 1, -1, -1);
        p.setPen(QPen(i == target ? QColor("#087bff") : palette().color(QPalette::Mid), i == target ? 2 : 1));
        p.setBrush(palette().base());
        p.drawRoundedRect(r, 9, 9);
        p.save();
        p.setClipRect(r);
        if (i == dragging)
            p.setOpacity(.32);
        auto area = r.adjusted(7, 7, -7, -footer);
        const auto &img = tiles[i].image;
        if (!img.isNull()) {
            auto size = img.size().scaled(area.size(), Qt::KeepAspectRatio);
            QRect dest(QPoint(), size);
            dest.moveCenter(area.center());
            p.drawImage(dest, img);
        } else {
            p.setPen(palette().color(QPalette::PlaceholderText));
            p.drawText(area, Qt::AlignCenter, tr("Image preview"));
        }
        p.setOpacity(1);
        p.setPen(palette().color(QPalette::Text));
        p.drawText(QRect(r.left() + 10, r.bottom() - footer + 3, r.width() - 20, footer - 5),
                   Qt::AlignVCenter,
                   fontMetrics().elidedText(tiles[i].name, Qt::ElideMiddle, r.width() - 20));
        p.restore();
        if (i == target) {
            const int x = target > dragging ? r.right() - 2 : r.left();
            p.fillRect(QRect(x, r.top() + 5, 3, r.height() - 10), QColor("#087bff"));
        }
    }
}
int ImageGallery::tileAt(QPoint pos) const {
    auto boxes = frames(width());
    for (int i = 0; i < boxes.size(); ++i)
        if (boxes[i].contains(pos))
            return i;
    return -1;
}
void ImageGallery::mousePressEvent(QMouseEvent *e) {
    if (e->button() == Qt::LeftButton) {
        press = e->position().toPoint();
        pressed = tileAt(press);
    }
}
void ImageGallery::mouseMoveEvent(QMouseEvent *e) {
    if (!(e->buttons() & Qt::LeftButton) || pressed < 0 ||
        (e->position().toPoint() - press).manhattanLength() < QApplication::startDragDistance())
        return;
    dragging = pressed;
    pressed = -1;
    auto drag = new QDrag(this);
    auto data = new QMimeData;
    data->setData(mime, tiles[dragging].uid.toUtf8());
    drag->setMimeData(data);
    auto pix =
        grab(frames(width())[dragging]).scaled(220, 220, Qt::KeepAspectRatio, Qt::SmoothTransformation);
    drag->setPixmap(pix);
    drag->setHotSpot(pix.rect().center());
    update();
    drag->exec(Qt::MoveAction);
    dragging = -1;
    target = -1;
    update();
    drag->deleteLater();
    // Emit after the native drag loop ends; a save may refresh/delete this card.
    const auto order = pendingOrder;
    pendingOrder = {};
    if (!order.isEmpty())
        emit reordered(order);
}
void ImageGallery::mouseReleaseEvent(QMouseEvent *e) {
    if (e->button() == Qt::LeftButton && pressed >= 0 && tileAt(e->position().toPoint()) == pressed)
        emit opened(tiles[pressed].path);
    pressed = -1;
}
void ImageGallery::dragEnterEvent(QDragEnterEvent *e) {
    if (e->source() == this && e->mimeData()->hasFormat(mime))
        e->acceptProposedAction();
}
void ImageGallery::dragMoveEvent(QDragMoveEvent *e) {
    if (e->source() != this)
        return;
    target = tileAt(e->position().toPoint());
    if (target == dragging)
        target = -1;
    e->acceptProposedAction();
    update();
}
void ImageGallery::dragLeaveEvent(QDragLeaveEvent *e) {
    target = -1;
    update();
    e->accept();
}
void ImageGallery::dropEvent(QDropEvent *e) {
    if (e->source() != this || dragging < 0 || target < 0 || target == dragging)
        return;
    QStringList images;
    for (const auto &tile : tiles)
        images.append(tile.uid);
    images.move(dragging, target);
    QJsonArray ordered;
    int i = 0;
    for (const auto &value : attachments) {
        const auto a = value.toObject();
        ordered.append(a["mimeType"].toString().startsWith("image/") ? images[i++] : a["uid"].toString());
    }
    e->acceptProposedAction();
    pendingOrder = ordered;
}
