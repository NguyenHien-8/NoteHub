#pragma once
#include <QImage>
#include <QJsonArray>
#include <QWidget>

class ImageGallery final : public QWidget {
    Q_OBJECT
  public:
    explicit ImageGallery(const QJsonArray &attachments, QWidget *parent = nullptr);
    bool hasHeightForWidth() const override {
        return true;
    }
    int heightForWidth(int width) const override;
    QSize sizeHint() const override {
        return {480, heightForWidth(480)};
    }
    QSize minimumSizeHint() const override {
        return {160, 80};
    }
    QVector<QRect> frames(int width) const;
  signals:
    void reordered(const QJsonArray &uids);
    void opened(const QString &path);

  protected:
    void paintEvent(QPaintEvent *) override;
    void resizeEvent(QResizeEvent *) override;
    void mousePressEvent(QMouseEvent *) override;
    void mouseMoveEvent(QMouseEvent *) override;
    void mouseReleaseEvent(QMouseEvent *) override;
    void dragEnterEvent(QDragEnterEvent *) override;
    void dragMoveEvent(QDragMoveEvent *) override;
    void dragLeaveEvent(QDragLeaveEvent *) override;
    void dropEvent(QDropEvent *) override;

  private:
    struct Tile {
        QString uid, path, name;
        QImage image;
    };
    QVector<Tile> tiles;
    QJsonArray attachments;
    QJsonArray pendingOrder;
    QPoint press;
    int pressed = -1, dragging = -1, target = -1;
    int tileAt(QPoint point) const;
};
