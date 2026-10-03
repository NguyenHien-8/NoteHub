#include "MainWindow.h"
#include "ImageGallery.h"
#include "NoteEditor.h"
#include <QTextList>
#include <QAbstractTextDocumentLayout>
#include <QCompleter>
#include <QIconEngine>
#include <QPixmapCache>
#include <QStyleHints>
#include <QtWidgets>
#include <cmath>
#include <memory>

namespace {
enum class Glyph { Sidebar, Home, Calendar, Search, Attachment, Tags, Settings, Folder, Star, Share, More,
                   Edit, Trash, Plus, Note, ChevronLeft, ChevronRight };

void paintGlyph(QPainter &p, Glyph glyph, const QColor &color, const QRectF &bounds, bool filled = false) {
    p.save();
    p.setRenderHint(QPainter::Antialiasing);
    const qreal scale = qMin(bounds.width(), bounds.height()) / 24.0;
    const QPointF offset(bounds.center().x() - 12.0 * scale, bounds.center().y() - 12.0 * scale);
    p.translate(offset);
    p.scale(scale, scale);
    constexpr qreal pi = 3.14159265358979323846;
    QPen pen(color, 1.85, Qt::SolidLine, Qt::RoundCap, Qt::RoundJoin);
    p.setPen(pen);
    p.setBrush(Qt::NoBrush);

    switch (glyph) {
    case Glyph::Sidebar:
        p.drawRoundedRect(QRectF(3.5, 4.5, 17, 15), 2, 2);
        p.drawLine(QPointF(9, 4.5), QPointF(9, 19.5));
        p.drawLine(QPointF(6.2, 8), QPointF(6.2, 16));
        break;
    case Glyph::Home: {
        QPainterPath path;
        path.moveTo(3.5, 11.5);
        path.lineTo(12, 4.5);
        path.lineTo(20.5, 11.5);
        path.moveTo(6, 10.5);
        path.lineTo(6, 20);
        path.lineTo(18, 20);
        path.lineTo(18, 10.5);
        p.drawPath(path);
        p.drawRoundedRect(QRectF(10, 14, 4, 6), 0.8, 0.8);
        break;
    }
    case Glyph::Calendar:
        p.drawRoundedRect(QRectF(4, 5.5, 16, 14.5), 2, 2);
        p.drawLine(QPointF(4, 9.5), QPointF(20, 9.5));
        p.drawLine(QPointF(8, 3.8), QPointF(8, 7));
        p.drawLine(QPointF(16, 3.8), QPointF(16, 7));
        for (qreal y : {13.0, 17.0})
            for (qreal x : {8.0, 12.0, 16.0})
                p.drawPoint(QPointF(x, y));
        break;
    case Glyph::Search:
        p.drawEllipse(QRectF(4, 4, 11.5, 11.5));
        p.drawLine(QPointF(14.5, 14.5), QPointF(20, 20));
        break;
    case Glyph::Attachment: {
        // A vertical paperclip is easier to recognise than the older chain-like mark.
        QPainterPath path;
        path.moveTo(8.4, 12.7);
        path.lineTo(14.9, 6.2);
        path.cubicTo(17.2, 3.9, 20.9, 7.5, 18.5, 9.9);
        path.lineTo(10.7, 17.7);
        path.cubicTo(7.1, 21.3, 1.9, 16.2, 5.5, 12.6);
        path.lineTo(13.4, 4.7);
        p.drawPath(path);
        break;
    }
    case Glyph::Tags: {
        QPainterPath path;
        path.moveTo(4, 5.5);
        path.lineTo(12.8, 5.5);
        path.lineTo(20, 12.6);
        path.lineTo(12.2, 20);
        path.lineTo(4, 11.8);
        path.closeSubpath();
        p.drawPath(path);
        p.drawEllipse(QRectF(7, 8.2, 2.3, 2.3));
        break;
    }
    case Glyph::Settings: {
        // Eight-tooth gear with a clear centre hole: reads as Settings in both themes.
        QPolygonF gear;
        constexpr int points = 32;
        for (int i = 0; i < points; ++i) {
            const qreal angle = -(pi / 2.0) + i * (2.0 * pi / points);
            const int phase = i % 4;
            const qreal radius = (phase == 0 || phase == 3) ? 9.2 : 7.3;
            gear << QPointF(12 + std::cos(angle) * radius, 12 + std::sin(angle) * radius);
        }
        p.drawPolygon(gear);
        p.drawEllipse(QRectF(9, 9, 6, 6));
        break;
    }
    case Glyph::Folder: {
        QPainterPath path;
        path.moveTo(3.5, 7);
        path.lineTo(9.2, 7);
        path.lineTo(11.3, 9.2);
        path.lineTo(20.5, 9.2);
        path.lineTo(20.5, 19);
        path.lineTo(3.5, 19);
        path.closeSubpath();
        p.drawPath(path);
        p.drawLine(QPointF(3.5, 11.2), QPointF(20.5, 11.2));
        break;
    }
    case Glyph::Star: {
        QPolygonF points;
        for (int i = 0; i < 10; ++i) {
            const qreal a = -(pi / 2.0) + i * pi / 5.0;
            const qreal r = i % 2 ? 4.0 : 9.0;
            points << QPointF(12 + std::cos(a) * r, 12 + std::sin(a) * r);
        }
        if (filled)
            p.setBrush(color);
        p.drawPolygon(points);
        break;
    }
    case Glyph::Share:
        p.drawEllipse(QRectF(4, 10, 4, 4));
        p.drawEllipse(QRectF(16, 5, 4, 4));
        p.drawEllipse(QRectF(16, 15, 4, 4));
        p.drawLine(QPointF(8, 11), QPointF(16, 8));
        p.drawLine(QPointF(8, 13), QPointF(16, 16));
        break;
    case Glyph::More:
        p.setBrush(color);
        p.setPen(Qt::NoPen);
        p.drawEllipse(QRectF(5, 11, 2, 2));
        p.drawEllipse(QRectF(11, 11, 2, 2));
        p.drawEllipse(QRectF(17, 11, 2, 2));
        break;
    case Glyph::Edit:
        p.drawLine(QPointF(5, 19), QPointF(8, 16));
        p.drawLine(QPointF(8, 16), QPointF(17, 7));
        p.drawLine(QPointF(17, 7), QPointF(20, 10));
        p.drawLine(QPointF(20, 10), QPointF(11, 19));
        p.drawLine(QPointF(5, 19), QPointF(11, 19));
        break;
    case Glyph::Trash:
        p.drawRoundedRect(QRectF(6, 8, 12, 12), 1.5, 1.5);
        p.drawLine(QPointF(4.5, 7), QPointF(19.5, 7));
        p.drawLine(QPointF(9, 4.5), QPointF(15, 4.5));
        p.drawLine(QPointF(10, 11), QPointF(10, 17));
        p.drawLine(QPointF(14, 11), QPointF(14, 17));
        break;
    case Glyph::Plus:
        p.drawLine(QPointF(12, 5), QPointF(12, 19));
        p.drawLine(QPointF(5, 12), QPointF(19, 12));
        break;
    case Glyph::Note:
        p.drawRoundedRect(QRectF(5, 4, 14, 16), 2, 2);
        p.drawLine(QPointF(8, 9), QPointF(16, 9));
        p.drawLine(QPointF(8, 13), QPointF(16, 13));
        p.drawLine(QPointF(8, 17), QPointF(13, 17));
        break;
    case Glyph::ChevronLeft:
        p.drawPolyline(QPolygonF{QPointF(15, 5), QPointF(8, 12), QPointF(15, 19)});
        break;
    case Glyph::ChevronRight:
        p.drawPolyline(QPolygonF{QPointF(9, 5), QPointF(16, 12), QPointF(9, 19)});
        break;
    }
    p.restore();
}

class PaletteIconEngine final : public QIconEngine {
  public:
    explicit PaletteIconEngine(Glyph value) : glyph(value) {}
    QIconEngine *clone() const override { return new PaletteIconEngine(glyph); }

    void paint(QPainter *painter, const QRect &rect, QIcon::Mode mode, QIcon::State state) override {
        const QPalette palette = qApp ? qApp->palette() : QPalette();
        QColor color;
        if (mode == QIcon::Disabled)
            color = palette.color(QPalette::Disabled, QPalette::Text);
        else if (state == QIcon::On || mode == QIcon::Selected)
            color = palette.color(QPalette::Highlight);
        else if (mode == QIcon::Active)
            color = palette.color(QPalette::Text);
        else
            color = palette.color(QPalette::PlaceholderText);
        paintGlyph(*painter, glyph, color, rect.adjusted(1, 1, -1, -1), state == QIcon::On && glyph == Glyph::Star);
    }

    QPixmap pixmap(const QSize &size, QIcon::Mode mode, QIcon::State state) override {
        QPixmap pix(size);
        pix.fill(Qt::transparent);
        QPainter painter(&pix);
        paint(&painter, pix.rect(), mode, state);
        return pix;
    }

  private:
    Glyph glyph;
};

QIcon lineIcon(Glyph glyph) {
    return QIcon(new PaletteIconEngine(glyph));
}

class ContainedTextBrowser final : public QTextBrowser {
  public:
    using QTextBrowser::QTextBrowser;

  protected:
    void wheelEvent(QWheelEvent *event) override {
        // The preview deliberately owns wheel input even though its scrollbar is
        // hidden. This avoids Qt chaining an exhausted child scroll area into the
        // outer timeline while the pointer is still over the same memo.
        if (event->modifiers().testFlag(Qt::ControlModifier)) {
            QTextBrowser::wheelEvent(event);
            event->accept();
            return;
        }
        if (auto bar = verticalScrollBar(); bar && bar->maximum() > bar->minimum()) {
            int delta = event->pixelDelta().y();
            if (delta == 0)
                delta = qRound((event->angleDelta().y() / 120.0) * QApplication::wheelScrollLines() *
                               qMax(12, fontMetrics().lineSpacing()));
            bar->setValue(bar->value() - delta);
        }
        event->accept();
    }
};

class WheelGuardFrame final : public QFrame {
  public:
    using QFrame::QFrame;

  protected:
    void wheelEvent(QWheelEvent *event) override {
        // Wheel events ignored by non-scrollable children stop at the memo card
        // instead of bubbling into the timeline's QScrollArea.
        event->accept();
    }
};

void paintChevron(QWidget *widget, bool open) {
    if (!widget)
        return;
    QPainter painter(widget);
    painter.setRenderHint(QPainter::Antialiasing);
    const auto group = widget->isEnabled() ? QPalette::Active : QPalette::Disabled;
    QColor color = widget->palette().color(group, QPalette::Text);
    QPen pen(color, 1.6, Qt::SolidLine, Qt::RoundCap, Qt::RoundJoin);
    painter.setPen(pen);
    const qreal cx = widget->width() - 15.0;
    const qreal cy = widget->height() / 2.0;
    QPolygonF arrow;
    if (open)
        arrow << QPointF(cx - 4.0, cy + 2.0) << QPointF(cx, cy - 2.0) << QPointF(cx + 4.0, cy + 2.0);
    else
        arrow << QPointF(cx - 4.0, cy - 2.0) << QPointF(cx, cy + 2.0) << QPointF(cx + 4.0, cy - 2.0);
    painter.drawPolyline(arrow);
}

class ArrowComboBox final : public QComboBox {
  public:
    using QComboBox::QComboBox;

  protected:
    void showPopup() override {
        popupOpen = true;
        QComboBox::showPopup();
        if (view() && view()->window())
            view()->window()->setFixedWidth(width());
        update();
    }
    void hidePopup() override {
        QComboBox::hidePopup();
        popupOpen = false;
        update();
    }
    void paintEvent(QPaintEvent *event) override {
        QComboBox::paintEvent(event);
        paintChevron(this, popupOpen);
    }

  private:
    bool popupOpen = false;
};

class ArrowFontComboBox final : public QFontComboBox {
  public:
    using QFontComboBox::QFontComboBox;

  protected:
    void showPopup() override {
        popupOpen = true;
        QFontComboBox::showPopup();
        if (view()) {
            view()->setMinimumWidth(width());
            view()->setMaximumWidth(width());
            if (view()->window())
                view()->window()->setFixedWidth(width());
        }
        update();
    }
    void hidePopup() override {
        QFontComboBox::hidePopup();
        popupOpen = false;
        update();
    }
    void paintEvent(QPaintEvent *event) override {
        QFontComboBox::paintEvent(event);
        paintChevron(this, popupOpen);
    }

  private:
    bool popupOpen = false;
};

class SpinStepButton final : public QToolButton {
  public:
    enum Direction { Up, Down };

    SpinStepButton(Direction direction, QWidget *parent = nullptr) : QToolButton(parent), direction(direction) {
        setObjectName("spinStepButton");
        setFocusPolicy(Qt::NoFocus);
        setCursor(Qt::ArrowCursor);
        setAutoRepeat(true);
        setAutoRepeatDelay(350);
        setAutoRepeatInterval(80);
        setToolTip(direction == Up ? QObject::tr("Increase") : QObject::tr("Decrease"));
    }

  protected:
    void paintEvent(QPaintEvent *event) override {
        QToolButton::paintEvent(event);
        QPainter painter(this);
        painter.setRenderHint(QPainter::Antialiasing);
        const auto group = isEnabled() ? QPalette::Active : QPalette::Disabled;
        const QColor color = palette().color(group, QPalette::Text);
        painter.setPen(QPen(color, 1.7, Qt::SolidLine, Qt::RoundCap, Qt::RoundJoin));
        const QPointF c = rect().center();
        const qreal dy = direction == Up ? 1.7 : -1.7;
        painter.drawPolyline(QPolygonF{QPointF(c.x() - 3.6, c.y() + dy),
                                      QPointF(c.x(), c.y() - dy),
                                      QPointF(c.x() + 3.6, c.y() + dy)});
    }

  private:
    Direction direction;
};

class ArrowSpinBox final : public QSpinBox {
  public:
    explicit ArrowSpinBox(QWidget *parent = nullptr) : QSpinBox(parent) {
        // Native QSpinBox sub-controls are style-dependent.  On some Windows
        // styles the lower half of a custom-painted spin box can still hit the
        // native UP sub-control.  Removing the native buttons and using two
        // explicit child buttons makes step-up / step-down deterministic.
        setButtonSymbols(QAbstractSpinBox::NoButtons);
        increase = new SpinStepButton(SpinStepButton::Up, this);
        decrease = new SpinStepButton(SpinStepButton::Down, this);
        increase->setObjectName("spinIncreaseButton");
        decrease->setObjectName("spinDecreaseButton");
        connect(increase, &QToolButton::clicked, this, &QAbstractSpinBox::stepUp);
        connect(decrease, &QToolButton::clicked, this, &QAbstractSpinBox::stepDown);
    }

  protected:
    void resizeEvent(QResizeEvent *event) override {
        QSpinBox::resizeEvent(event);
        constexpr int buttonWidth = 30;
        const int half = height() / 2;
        increase->setGeometry(width() - buttonWidth - 1, 1, buttonWidth, qMax(1, half - 1));
        decrease->setGeometry(width() - buttonWidth - 1, half, buttonWidth, qMax(1, height() - half - 1));
        increase->raise();
        decrease->raise();
    }

  private:
    SpinStepButton *increase = nullptr;
    SpinStepButton *decrease = nullptr;
};

void applyCalendarChrome(QCalendarWidget *widget) {
    if (!widget)
        return;
    if (auto previous = widget->findChild<QToolButton *>("qt_calendar_prevmonth")) {
        previous->setIcon(lineIcon(Glyph::ChevronLeft));
        previous->setIconSize({18, 18});
        previous->setToolTip(QObject::tr("Previous month"));
    }
    if (auto next = widget->findChild<QToolButton *>("qt_calendar_nextmonth")) {
        next->setIcon(lineIcon(Glyph::ChevronRight));
        next->setIconSize({18, 18});
        next->setToolTip(QObject::tr("Next month"));
    }
    const QPalette appPalette = qApp->palette();
    QPalette calendarPalette = widget->palette();
    for (auto role : {QPalette::Window, QPalette::Base, QPalette::AlternateBase, QPalette::Button})
        calendarPalette.setColor(role, appPalette.color(QPalette::Base));
    calendarPalette.setColor(QPalette::WindowText, appPalette.color(QPalette::Text));
    calendarPalette.setColor(QPalette::Text, appPalette.color(QPalette::Text));
    calendarPalette.setColor(QPalette::ButtonText, appPalette.color(QPalette::Text));
    calendarPalette.setColor(QPalette::Highlight, appPalette.color(QPalette::Highlight));
    calendarPalette.setColor(QPalette::HighlightedText, appPalette.color(QPalette::HighlightedText));
    calendarPalette.setColor(QPalette::PlaceholderText, appPalette.color(QPalette::PlaceholderText));
    widget->setPalette(calendarPalette);
    for (auto view : widget->findChildren<QAbstractItemView *>()) {
        view->setPalette(calendarPalette);
        view->viewport()->setPalette(calendarPalette);
        view->viewport()->setAutoFillBackground(true);
    }
    widget->update();
}

void applyCalendarDateColors(QCalendarWidget *widget, int year, int month) {
    if (!widget)
        return;
    widget->setDateTextFormat(QDate(), QTextCharFormat());

    const QColor text = qApp->palette().color(QPalette::Text);
    const QColor muted = qApp->palette().color(QPalette::PlaceholderText);
    const bool dark = qApp->palette().color(QPalette::Window).lightness() < 128;
    const QColor weekend = dark ? QColor("#ff7b7b") : QColor("#d92d20");

    QTextCharFormat weekdayHeader;
    weekdayHeader.setForeground(text);
    for (auto day : {Qt::Monday, Qt::Tuesday, Qt::Wednesday, Qt::Thursday, Qt::Friday})
        widget->setWeekdayTextFormat(day, weekdayHeader);
    QTextCharFormat weekendHeader;
    weekendHeader.setForeground(weekend);
    widget->setWeekdayTextFormat(Qt::Saturday, weekendHeader);
    widget->setWeekdayTextFormat(Qt::Sunday, weekendHeader);

    const QDate first(year, month, 1);
    if (!first.isValid())
        return;
    const int firstWeekday = int(widget->firstDayOfWeek());
    const int leading = (first.dayOfWeek() - firstWeekday + 7) % 7;
    QDate day = first.addDays(-leading);
    for (int i = 0; i < 42; ++i, day = day.addDays(1)) {
        QTextCharFormat format;
        if (day.month() != month || day.year() != year)
            format.setForeground(muted);
        else if (day.dayOfWeek() == Qt::Saturday || day.dayOfWeek() == Qt::Sunday)
            format.setForeground(weekend);
        else
            format.setForeground(text);
        widget->setDateTextFormat(day, format);
    }
}

QPushButton *button(const QString &text, QWidget *parent = nullptr) {
    auto b = new QPushButton(text, parent);
    b->setCursor(Qt::PointingHandCursor);
    return b;
}

QToolButton *iconButton(Glyph glyph, const QString &tip, QWidget *parent = nullptr) {
    auto b = new QToolButton(parent);
    b->setIcon(lineIcon(glyph));
    b->setIconSize({20, 20});
    b->setToolTip(tip);
    b->setCursor(Qt::PointingHandCursor);
    b->setAutoRaise(true);
    b->setObjectName("iconButton");
    return b;
}

QFrame *card(const char *name = "card") {
    QFrame *f = qstrcmp(name, "noteCard") == 0 ? static_cast<QFrame *>(new WheelGuardFrame)
                                                : static_cast<QFrame *>(new QFrame);
    f->setObjectName(name);
    f->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    return f;
}

QJsonArray jsonPaths(const QStringList &paths) {
    QJsonArray a;
    for (const auto &p : paths)
        a.append(p);
    return a;
}
QStringList fileFailures(const QJsonValue &result) {
    QStringList list;
    for (const auto &v : result.toObject()["failures"].toArray())
        list.append(v.toString());
    return list;
}
class MemoDialog final : public QDialog {
  public:
    using QDialog::QDialog;
    NoteEditor *editor = nullptr;
    bool busy = false;
    void reject() override {
        if (busy)
            return;
        if (editor && editor->modified() &&
            QMessageBox::question(
                this, tr("Discard edits?"),
                tr("Unsaved text will be lost. Confirmed attachment changes stay saved.")) !=
                QMessageBox::Yes)
            return;
        QDialog::reject();
    }
};
} // namespace

MainWindow::MainWindow(const QString &program, const QString &profile, QWidget *parent)
    : QMainWindow(parent), backend(this) {
    setWindowTitle("NoteHub");
    resize(1360, 840);
    setMinimumSize(760, 520);
    statusBar()->setSizeGripEnabled(false);
    statusBar()->hide();

    auto root = new QWidget(this);
    root->setObjectName("appRoot");
    setCentralWidget(root);
    auto outer = new QVBoxLayout(root);
    outer->setContentsMargins(18, 14, 18, 14);
    outer->setSpacing(12);

    auto header = new QHBoxLayout;
    header->setSpacing(10);
    leftToggle = button({});
    leftToggle->setIcon(lineIcon(Glyph::Sidebar));
    leftToggle->setIconSize({21, 21});
    leftToggle->setFixedSize(38, 38);
    leftToggle->setObjectName("headerIconButton");
    leftToggle->setToolTip(tr("Collapse navigation"));
    header->addWidget(leftToggle);

    auto logo = new QLabel;
    logo->setFixedSize(42, 42);
    logo->setPixmap(QPixmap(":/icons/NoteHub.png").scaled(38, 38, Qt::KeepAspectRatio, Qt::SmoothTransformation));
    logo->setAlignment(Qt::AlignCenter);
    header->addWidget(logo);
    auto brand = new QLabel("NoteHub");
    brand->setObjectName("brand");
    header->addWidget(brand);
    header->addStretch(1);

    search = new QLineEdit;
    search->setObjectName("globalSearch");
    search->setPlaceholderText(tr("Search notes, tags and content…"));
    search->setClearButtonEnabled(true);
    search->setMaximumWidth(480);
    search->setMinimumWidth(220);
    search->setFixedHeight(42);
    search->addAction(lineIcon(Glyph::Search), QLineEdit::LeadingPosition);
    search->setToolTip(tr("Search · Ctrl+K"));
    header->addWidget(search, 1);
    outer->addLayout(header);

    navigationSplitter = new QSplitter(Qt::Horizontal);
    navigationSplitter->setObjectName("navigationSplitter");
    navigationSplitter->setChildrenCollapsible(false);
    navigationSplitter->setHandleWidth(7);
    outer->addWidget(navigationSplitter, 1);

    auto contentColumns = new QWidget;
    contentColumns->setObjectName("contentColumns");
    auto columns = new QHBoxLayout(contentColumns);
    columns->setContentsMargins(0, 0, 0, 0);
    columns->setSpacing(10);

    navigation = new QFrame;
    navigation->setObjectName("sideRail");
    auto nav = new QVBoxLayout(navigation);
    nav->setSpacing(5);
    nav->setContentsMargins(7, 9, 7, 9);
    struct NavEntry { QString label; Glyph glyph; QString destination; };
    const QList<NavEntry> entries = {{"Home", Glyph::Home, "home"},
                                     {"Calendar", Glyph::Calendar, "calendar"},
                                     {"Search", Glyph::Search, "search"},
                                     {"Attachments", Glyph::Attachment, "attachments"},
                                     {"Tags", Glyph::Tags, "tags"},
                                     {"Settings", Glyph::Settings, "settings"}};
    for (const auto &entry : entries) {
        auto b = new QToolButton;
        b->setObjectName("navButton");
        b->setText(entry.label);
        b->setProperty("label", entry.destination);
        b->setToolTip(entry.label);
        b->setAccessibleName(entry.label);
        b->setAccessibleDescription(tr("Navigate to %1").arg(entry.label));
        b->setIcon(lineIcon(entry.glyph));
        b->setIconSize({21, 21});
        b->setCheckable(true);
        b->setAutoExclusive(true);
        b->setMinimumHeight(44);
        b->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
        b->setCursor(Qt::PointingHandCursor);
        nav->addWidget(b);
        navButtons.append(b);
        connect(b, &QToolButton::clicked, this, [this, destination = entry.destination] { navigate(destination); });
    }
    auto separator = new QFrame;
    separator->setObjectName("railSeparator");
    separator->setFrameShape(QFrame::HLine);
    nav->addWidget(separator);
    auto myTags = new QToolButton;
    myTags->setObjectName("navButton");
    myTags->setProperty("label", "my tags");
    myTags->setText(tr("My Tags"));
    myTags->setToolTip(tr("My Tags"));
    myTags->setAccessibleName(tr("My Tags"));
    myTags->setIcon(lineIcon(Glyph::Folder));
    myTags->setIconSize({21, 21});
    myTags->setMinimumHeight(44);
    myTags->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
    myTags->setCursor(Qt::PointingHandCursor);
    nav->addWidget(myTags);
    navButtons.append(myTags);
    nav->addStretch();
    connect(myTags, &QToolButton::clicked, this, [this, myTags] {
        QMenu menu;
        menu.setObjectName("tagMenu");
        for (const auto &v : tags) {
            auto t = v.toObject();
            auto a = menu.addAction("#" + t["tag"].toString() + "    " + QString::number(t["count"].toInt()));
            connect(a, &QAction::triggered, this, [this, t] {
                tag = t["tag"].toString();
                page = "home";
                refresh();
            });
        }
        if (!tags.isEmpty())
            menu.addSeparator();
        auto add = menu.addAction(tr("Add tag to draft…"));
        connect(add, &QAction::triggered, this, [this] {
            bool ok = false;
            auto value = QInputDialog::getText(this, tr("Add tag"), tr("Tag (for example Research/FPGA)"),
                                               QLineEdit::Normal, {}, &ok);
            if (ok && !value.trimmed().isEmpty()) {
                navigate("home");
                composer->insertPlainText(" #" + value.trimmed().remove('#'));
                composer->setFocus();
            }
        });
        menu.exec(myTags->mapToGlobal(QPoint(myTags->width(), 0)));
    });

    navigationScroll = new QScrollArea;
    navigationScroll->setWidgetResizable(true);
    navigationScroll->setFrameShape(QFrame::NoFrame);
    navigationScroll->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    navigationScroll->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    navigationScroll->setWidget(navigation);
    navigationScroll->setObjectName("navScroll");
    navigationScroll->setMinimumWidth(64);
    navigationScroll->setMaximumWidth(280);
    navigationSplitter->addWidget(navigationScroll);
    navigationSplitter->addWidget(contentColumns);
    navigationSplitter->setStretchFactor(0, 0);
    navigationSplitter->setStretchFactor(1, 1);
    navigationSplitter->setCollapsible(0, false);
    navigationSplitter->setCollapsible(1, false);
    if (auto handle = navigationSplitter->handle(1)) {
        handle->setToolTip(tr("Drag to resize the navigation sidebar"));
        handle->setAccessibleName(tr("Resize navigation sidebar"));
    }

    workspace = new QFrame;
    workspace->setObjectName("workspace");
    auto centerLayout = new QVBoxLayout(workspace);
    centerLayout->setContentsMargins(12, 12, 12, 12);
    centerLayout->setSpacing(10);
    columns->addWidget(workspace, 1);

    composerPanel = card("composerCard");
    auto compose = new QVBoxLayout(composerPanel);
    compose->setContentsMargins(11, 10, 11, 9);
    compose->setSpacing(7);
    composer = new QTextEdit;
    composer->setObjectName("composer");
    composer->setAcceptRichText(false);
    composer->setPlaceholderText(tr("Có suy nghĩ gì…"));
    composer->setMinimumHeight(72);
    composer->setMaximumHeight(105);
    compose->addWidget(composer);
    stagedLabel = new QLabel;
    stagedLabel->setObjectName("stagedFiles");
    stagedLabel->setWordWrap(true);
    stagedLabel->hide();
    compose->addWidget(stagedLabel);
    auto actions = new QHBoxLayout;
    actions->setSpacing(4);
    auto attach = button(tr("Attach files"));
    attach->setObjectName("composerAction");
    attach->setIcon(lineIcon(Glyph::Attachment));
    auto addTag = button(tr("Add tag"));
    addTag->setObjectName("composerAction");
    addTag->setIcon(lineIcon(Glyph::Plus));
    clearStage = button(tr("Clear files"));
    clearStage->setObjectName("subtleButton");
    clearStage->hide();
    actions->addWidget(attach);
    actions->addWidget(addTag);
    actions->addWidget(clearStage);
    actions->addStretch();
    save = button(tr("Save"));
    save->setObjectName("primary");
    save->setMinimumWidth(72);
    actions->addWidget(save);
    compose->addLayout(actions);
    centerLayout->addWidget(composerPanel);

    connect(attach, &QPushButton::clicked, this, [this] {
        const auto selected = QFileDialog::getOpenFileNames(this, tr("Attach files"));
        for (const auto &p : selected)
            if (!staged.contains(p))
                staged.append(p);
        updateStagedSummary();
    });
    connect(addTag, &QPushButton::clicked, this, [this] {
        bool ok = false;
        auto value = QInputDialog::getText(this, tr("Add tag"), tr("Tag (for example Research/FPGA)"),
                                           QLineEdit::Normal, {}, &ok);
        if (ok && !value.trimmed().isEmpty()) {
            composer->insertPlainText(" #" + value.trimmed().remove('#'));
            composer->setFocus();
        }
    });
    connect(clearStage, &QPushButton::clicked, this, [this] {
        staged.clear();
        updateStagedSummary();
    });
    connect(save, &QPushButton::clicked, this, [this] {
        if (composer->toPlainText().trimmed().isEmpty() && staged.isEmpty())
            return;
        const auto text = composer->toPlainText();
        const auto files = staged;
        composerPanel->setEnabled(false);
        backend.call(this, "memos.create", {{"content", text}},
                     [this, files](const QJsonValue &value, const QString &code, const QString &message) {
                         composerPanel->setEnabled(true);
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         composer->clear();
                         staged.clear();
                         updateStagedSummary();
                         const auto uid = value.toObject()["uid"].toString();
                         if (files.isEmpty()) {
                             refresh();
                             metadata();
                         } else
                             addFiles(uid, files, [this] { refresh(); metadata(); });
                     });
    });

    auto titleRow = new QHBoxLayout;
    heading = new QLabel(tr("All Notes"));
    heading->setObjectName("heading");
    titleRow->addWidget(heading);
    titleRow->addStretch();
    auto clear = button(tr("Clear filters"));
    clear->setObjectName("subtleButton");
    titleRow->addWidget(clear);
    centerLayout->addLayout(titleRow);
    connect(clear, &QPushButton::clicked, this, [this] {
        tag.clear();
        date.clear();
        search->clear();
        navigate("home");
    });

    scroll = new QScrollArea;
    scroll->setObjectName("contentScroll");
    scroll->setWidgetResizable(true);
    scroll->setFrameShape(QFrame::NoFrame);
    scroll->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    rows = new QWidget;
    rows->setObjectName("rows");
    rowLayout = new QVBoxLayout(rows);
    rowLayout->setContentsMargins(0, 0, 3, 0);
    rowLayout->setSpacing(10);
    rowLayout->setAlignment(Qt::AlignTop);
    scroll->setWidget(rows);
    centerLayout->addWidget(scroll, 1);
    more = button(tr("Load more"));
    more->setObjectName("outlineButton");
    more->hide();
    centerLayout->addWidget(more, 0, Qt::AlignHCenter);
    connect(more, &QPushButton::clicked, this, [this] { refresh(true); });

    rightPanel = new QWidget;
    rightPanel->setObjectName("rightColumn");
    rightPanel->setFixedWidth(300);
    auto side = new QVBoxLayout(rightPanel);
    side->setContentsMargins(0, 0, 0, 0);
    side->setSpacing(12);
    auto calendarCard = card("sideCard");
    auto calendarLayout = new QVBoxLayout(calendarCard);
    calendarLayout->setContentsMargins(10, 9, 10, 10);
    calendar = new QCalendarWidget;
    calendar->setObjectName("miniCalendar");
    calendar->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
    calendar->setGridVisible(false);
    calendar->setFirstDayOfWeek(Qt::Monday);
    calendarLayout->addWidget(calendar);
    side->addWidget(calendarCard);
    connect(calendar, &QCalendarWidget::clicked, this, [this](QDate selected) {
        date = selected.toString(Qt::ISODate);
        page = "home";
        refresh();
    });
    connect(calendar, &QCalendarWidget::currentPageChanged, this, [this] { refreshCalendar(calendar); });

    auto quickCard = card("sideCard");
    auto quick = new QVBoxLayout(quickCard);
    quick->setContentsMargins(12, 12, 12, 12);
    quick->setSpacing(5);
    auto quickTitle = new QLabel(tr("Quick Filters"));
    quickTitle->setObjectName("sideTitle");
    quick->addWidget(quickTitle);
    auto addQuick = [this, quick](const QString &label, Glyph glyph, const QString &destination, QLabel *&count) {
        auto row = new QFrame;
        row->setObjectName("quickRow");
        auto layout = new QHBoxLayout(row);
        layout->setContentsMargins(6, 2, 6, 2);
        layout->setSpacing(6);
        auto action = new QToolButton;
        action->setObjectName("quickButton");
        action->setText(label);
        action->setIcon(lineIcon(glyph));
        action->setIconSize({18, 18});
        action->setToolButtonStyle(Qt::ToolButtonTextBesideIcon);
        action->setCursor(Qt::PointingHandCursor);
        action->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
        count = new QLabel("0");
        count->setObjectName("countBadge");
        count->setAlignment(Qt::AlignCenter);
        layout->addWidget(action, 1);
        layout->addWidget(count);
        quick->addWidget(row);
        connect(action, &QToolButton::clicked, this, [this, destination] { navigate(destination); });
    };
    addQuick(tr("All Notes"), Glyph::Note, "home", allCount);
    addQuick(tr("Favorites"), Glyph::Star, "favorites", favoriteCount);
    addQuick(tr("Shared"), Glyph::Share, "shared", sharedCount);
    side->addWidget(quickCard);
    side->addStretch();
    columns->addWidget(rightPanel);

    compact = settings.value("navigation/compact", true).toBool();
    const int savedNavigationWidth = settings.value(
        "navigation/width", compact ? 68 : settings.value("navigation/expandedWidth", 212).toInt()).toInt();
    navigationSplitter->setSizes({qBound(64, savedNavigationWidth, 280), 1200});
    connect(navigationSplitter, &QSplitter::splitterMoved, this, [this](int, int) {
        const int width = navigationScroll->width();
        compact = width < 136;
        settings.setValue("navigation/compact", compact);
        settings.setValue("navigation/width", width);
        if (!compact)
            settings.setValue("navigation/expandedWidth", width);
        applyNavigation();
    });
    connect(leftToggle, &QPushButton::clicked, this, [this] {
        const bool currentlyCompact = navigationScroll->width() < 136;
        const int target = currentlyCompact
                               ? qBound(160, settings.value("navigation/expandedWidth", 212).toInt(), 280)
                               : 68;
        const int remaining = qMax(420, navigationSplitter->width() - target - navigationSplitter->handleWidth());
        navigationSplitter->setSizes({target, remaining});
        compact = target < 136;
        settings.setValue("navigation/compact", compact);
        settings.setValue("navigation/width", target);
        applyNavigation();
    });
    debounce.setSingleShot(true);
    debounce.setInterval(300);
    connect(search, &QLineEdit::textChanged, this, [this] { debounce.start(); });
    connect(&debounce, &QTimer::timeout, this, [this] { navigate(search->text().isEmpty() ? "home" : "search"); });
    auto focus = new QShortcut(QKeySequence("Ctrl+K"), this);
    connect(focus, &QShortcut::activated, search, qOverload<>(&QWidget::setFocus));
    connect(&backend, &BackendClient::connected, this, [this](const QJsonObject &hello) {
        dataPath = hello["dataDir"].toString();
        save->setEnabled(true);
        metadata();
        refresh();
    });
    connect(&backend, &BackendClient::failed, this, [this](const QString &message) {
        save->setEnabled(false);
        showError(message);
    });
    save->setEnabled(false);
    connect(QGuiApplication::styleHints(), &QStyleHints::colorSchemeChanged, this, [this](Qt::ColorScheme) {
        if (settings.value("appearance", "Light").toString() == "System")
            applyAppearance();
    });
    applyAppearance();
    applyNavigation();
    QTimer::singleShot(0, this, [this, program, profile] { backend.start(program, profile); });
}

void MainWindow::updateStagedSummary() {
    if (staged.isEmpty()) {
        stagedLabel->hide();
        clearStage->hide();
        return;
    }
    QStringList names;
    for (const auto &p : staged)
        names.append(QFileInfo(p).fileName());
    stagedLabel->setText(tr("%1 file(s) ready to attach").arg(staged.size()));
    stagedLabel->setToolTip(names.join('\n'));
    stagedLabel->show();
    clearStage->show();
}

void MainWindow::showError(const QString &message) {
    if (activeError) {
        if (activeError->text() != message)
            activeError->setDetailedText(message);
        return;
    }
    activeError = new QMessageBox(QMessageBox::Warning, tr("NoteHub"), message, QMessageBox::Ok, this);
    activeError->setAttribute(Qt::WA_DeleteOnClose);
    activeError->open();
}
void MainWindow::rpc(const QString &method, const QJsonObject &params,
                     std::function<void(const QJsonValue &)> done) {
    backend.call(this, method, params,
                 [this, done](const QJsonValue &v, const QString &code, const QString &message) {
                     if (!code.isEmpty()) {
                         showError(message);
                         return;
                     }
                     if (done)
                         done(v);
                 });
}
void MainWindow::clearRows() {
    while (auto item = rowLayout->takeAt(0)) {
        if (auto w = item->widget())
            w->deleteLater();
        delete item;
    }
}
void MainWindow::navigate(const QString &destination) {
    debounce.stop();
    if (destination != "search") {
        QSignalBlocker blocker(search);
        search->clear();
    }
    page = destination;
    tag.clear();
    date.clear();
    cursor.clear();
    searchOffset = 0;
    attachmentOffset = 0;
    applyNavigation();
    refresh();
}
void MainWindow::applyNavigation() {
    int railWidth = navigationScroll ? navigationScroll->width() : 0;
    if (railWidth <= 0)
        railWidth = compact ? 68 : settings.value("navigation/expandedWidth", 212).toInt();
    const bool small = railWidth < 136;
    compact = small;
    leftToggle->setToolTip(small ? tr("Expand navigation") : tr("Collapse navigation"));
    const QString active = (page == "favorites" || page == "shared") ? "home" : page;
    for (auto b : navButtons) {
        b->setToolButtonStyle(small ? Qt::ToolButtonIconOnly : Qt::ToolButtonTextBesideIcon);
        if (b->property("compact").toBool() != small) {
            b->setProperty("compact", small);
            b->style()->unpolish(b);
            b->style()->polish(b);
        }
        const auto key = b->property("label").toString();
        if (b->isCheckable())
            b->setChecked(key == active);
    }
    rightPanel->setVisible(width() >= 1180);
}
void MainWindow::resizeEvent(QResizeEvent *e) {
    QMainWindow::resizeEvent(e);
    applyNavigation();
}
void MainWindow::metadata() {
    rpc("metadata", {}, [this](const QJsonValue &v) {
        auto m = v.toObject();
        tags = m["tags"].toArray();
        allCount->setText(QString::number(m["all"].toInt()));
        favoriteCount->setText(QString::number(m["favorites"].toInt()));
        sharedCount->setText(QString::number(m["shared"].toInt()));
    });
    refreshCalendar(calendar);
}
void MainWindow::refreshCalendar(QCalendarWidget *widget) {
    if (!widget)
        return;
    const int year = widget->yearShown(), month = widget->monthShown();
    applyCalendarChrome(widget);
    applyCalendarDateColors(widget, year, month);
    if (!backend.isReady())
        return;

    backend.call(widget, "calendar.month", {{"year", year}, {"month", month}},
        [widget, year, month](const QJsonValue &v, const QString &code, const QString &) {
            if (!code.isEmpty() || widget->yearShown() != year || widget->monthShown() != month)
                return;

            // Keep all base date colors theme-aware, then use the accent only
            // for days that contain notes.
            applyCalendarDateColors(widget, year, month);
            const QColor accent = qApp->palette().color(QPalette::Highlight);
            for (const auto &d : v.toArray()) {
                const auto day = d.toObject();
                const QDate date = QDate::fromString(day["date"].toString(), Qt::ISODate);
                auto format = widget->dateTextFormat(date);
                format.setFontWeight(QFont::DemiBold);
                format.setForeground(accent);
                format.setToolTip(QObject::tr("%1 notes").arg(day["count"].toInt()));
                widget->setDateTextFormat(date, format);
            }
            widget->update();
        });
}
void MainWindow::refresh(bool append, const QString &anchorUid) {
    if (!backend.isReady())
        return;
    if (append && loading)
        return;
    applyNavigation();

    // Preserve the edited card's viewport position across an asynchronous reload.
    // Rebuilding the timeline used to reset the scroll bar to the first memo.
    int restoreScroll = -1;
    int anchorViewportY = 0;
    bool anchorFound = false;
    if (!append && !anchorUid.isEmpty()) {
        restoreScroll = scroll->verticalScrollBar()->value();
        for (int i = 0; i < rowLayout->count(); ++i) {
            auto widget = rowLayout->itemAt(i)->widget();
            if (widget && widget->property("memoUid").toString() == anchorUid) {
                anchorViewportY = widget->mapTo(scroll->viewport(), QPoint(0, 0)).y();
                anchorFound = true;
                break;
            }
        }
    }

    const int current = ++generation;
    loading = true;
    more->hide();
    composerPanel->setVisible(page == "home");

    QString title;
    if (!tag.isEmpty())
        title = "#" + tag;
    else if (!date.isEmpty())
        title = QDate::fromString(date, Qt::ISODate).toString("dd MMMM yyyy");
    else if (page == "home")
        title = tr("All Notes");
    else if (page == "favorites")
        title = tr("Favorites");
    else if (page == "shared")
        title = tr("Shared");
    else if (page == "attachments")
        title = tr("Attachments");
    else if (page == "settings")
        title = tr("Settings");
    else if (page == "calendar")
        title = tr("Calendar");
    else if (page == "tags")
        title = tr("Tags");
    else
        title = tr("Search");
    heading->setText(title);

    if (!append) {
        cursor.clear();
        searchOffset = 0;
        attachmentOffset = 0;
        clearRows();
    }
    if (page == "settings") {
        loading = false;
        settingsPage();
        return;
    }
    if (page == "calendar") {
        loading = false;
        auto holder = card("contentCard");
        auto layout = new QVBoxLayout(holder);
        layout->setContentsMargins(14, 14, 14, 14);
        auto c = new QCalendarWidget;
        c->setObjectName("largeCalendar");
        c->setMinimumHeight(420);
        c->setFirstDayOfWeek(Qt::Monday);
        c->setVerticalHeaderFormat(QCalendarWidget::NoVerticalHeader);
        c->setGridVisible(false);
        layout->addWidget(c);
        rowLayout->addWidget(holder);
        connect(c, &QCalendarWidget::currentPageChanged, this, [this, c] { refreshCalendar(c); });
        refreshCalendar(c);
        connect(c, &QCalendarWidget::clicked, this, [this](QDate d) {
            date = d.toString(Qt::ISODate);
            page = "home";
            refresh();
        });
        return;
    }
    if (page == "tags") {
        loading = false;
        auto holder = card("contentCard");
        auto layout = new QVBoxLayout(holder);
        layout->setContentsMargins(16, 14, 16, 16);
        layout->setSpacing(12);
        auto hint = new QLabel(tr("Browse your tags and jump straight to the matching notes."));
        hint->setObjectName("muted");
        layout->addWidget(hint);
        auto gridHost = new QWidget;
        auto grid = new QGridLayout(gridHost);
        grid->setContentsMargins(0, 0, 0, 0);
        grid->setHorizontalSpacing(8);
        grid->setVerticalSpacing(8);
        layout->addWidget(gridHost);
        rowLayout->addWidget(holder);
        rpc("metadata", {}, [this, current, grid](const QJsonValue &v) {
            if (current != generation)
                return;
            tags = v.toObject()["tags"].toArray();
            int index = 0;
            const int columns = width() >= 1180 ? 3 : 2;
            for (const auto &item : tags) {
                auto t = item.toObject();
                auto b = button("#" + t["tag"].toString() + "     " + QString::number(t["count"].toInt()));
                b->setObjectName("tagCard");
                b->setMinimumHeight(48);
                b->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Fixed);
                grid->addWidget(b, index / columns, index % columns);
                ++index;
                connect(b, &QPushButton::clicked, this, [this, t] {
                    tag = t["tag"].toString();
                    page = "home";
                    refresh();
                });
            }
            if (tags.isEmpty()) {
                auto empty = new QLabel(tr("No tags yet. Add #tags to a note to organize it."));
                empty->setObjectName("emptyState");
                empty->setAlignment(Qt::AlignCenter);
                empty->setMinimumHeight(120);
                grid->addWidget(empty, 0, 0, 1, columns);
            }
        });
        return;
    }
    if (page == "attachments") {
        backend.call(this, "attachments.list", {{"limit", 40}, {"offset", attachmentOffset}},
                     [this, current](const QJsonValue &v, const QString &code, const QString &message) {
                         if (current != generation)
                             return;
                         loading = false;
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         auto values = v.toArray();
                         attachmentOffset += values.size();
                         more->setVisible(values.size() == 40);
                         for (const auto &item : values) {
                             auto a = item.toObject();
                             auto f = card("attachmentRow");
                             auto l = new QHBoxLayout(f);
                             l->setContentsMargins(12, 10, 10, 10);
                             l->setSpacing(10);
                             auto icon = new QToolButton;
                             icon->setObjectName("decorativeIcon");
                             icon->setIcon(lineIcon(Glyph::Attachment));
                             icon->setIconSize({22, 22});
                             icon->setFixedSize(30, 30);
                             icon->setAutoRaise(true);
                             icon->setFocusPolicy(Qt::NoFocus);
                             icon->setAttribute(Qt::WA_TransparentForMouseEvents);
                             l->addWidget(icon);
                             auto info = new QVBoxLayout;
                             info->setSpacing(1);
                             auto name = button(a["filename"].toString());
                             name->setObjectName("fileLink");
                             name->setToolTip(a["path"].toString());
                             name->setSizePolicy(QSizePolicy::Ignored, QSizePolicy::Preferred);
                             auto path = new QLabel(QDir::toNativeSeparators(a["path"].toString()));
                             path->setObjectName("muted");
                             path->setSizePolicy(QSizePolicy::Ignored, QSizePolicy::Preferred);
                             path->setToolTip(a["path"].toString());
                             path->setTextInteractionFlags(Qt::TextSelectableByMouse);
                             info->addWidget(name);
                             info->addWidget(path);
                             l->addLayout(info, 1);
                             auto note = button(tr("Open note"));
                             note->setObjectName("outlineButton");
                             auto remove = iconButton(Glyph::Trash, tr("Delete attachment"));
                             remove->setObjectName("dangerIconButton");
                             l->addWidget(note);
                             l->addWidget(remove);
                             rowLayout->addWidget(f);
                             connect(name, &QPushButton::clicked, this, [this, a] {
                                 if (!QDesktopServices::openUrl(QUrl::fromLocalFile(a["path"].toString())))
                                     showError(tr("Could not open attachment."));
                             });
                             connect(note, &QPushButton::clicked, this,
                                     [this, a] { editNote(a["memoUid"].toString()); });
                             connect(remove, &QToolButton::clicked, this, [this, a] {
                                 if (QMessageBox::question(this, tr("Delete attachment?"),
                                                           a["filename"].toString()) == QMessageBox::Yes)
                                     rpc("attachments.delete", {{"uid", a["uid"]}},
                                         [this](const QJsonValue &) { refresh(); metadata(); });
                             });
                         }
                         if (values.isEmpty() && attachmentOffset == 0) {
                             auto empty = card("contentCard");
                             auto layout = new QVBoxLayout(empty);
                             auto label = new QLabel(tr("No attachments yet\nFiles you attach to notes will appear here."));
                             label->setObjectName("emptyState");
                             label->setAlignment(Qt::AlignCenter);
                             label->setMinimumHeight(150);
                             layout->addWidget(label);
                             rowLayout->addWidget(empty);
                         }
                     });
        return;
    }
    QJsonObject p{{"limit", 20},
                  {"cursor", cursor},
                  {"offset", searchOffset},
                  {"favorite", page == "favorites"},
                  {"shared", page == "shared"}};
    if (!tag.isEmpty())
        p["tags"] = QJsonArray{tag};
    if (!date.isEmpty())
        p["date"] = date;
    if (page == "search")
        p["text"] = search->text();
    backend.call(
        this, "memos.list", p,
        [this, current, append, anchorUid, anchorViewportY, anchorFound, restoreScroll](const QJsonValue &v,
                                                                                       const QString &code,
                                                                                       const QString &message) {
            if (current != generation)
                return;
            loading = false;
            if (!code.isEmpty()) {
                showError(message);
                return;
            }
            auto result = v.toObject();
            auto items = result["items"].toArray();
            cursor = result["cursor"].toString();
            searchOffset = result["nextOffset"].toInt();
            more->setVisible(!cursor.isEmpty() || searchOffset > 0);
            for (const auto &item : items)
                addNote(item.toObject());
            if (items.isEmpty() && !append) {
                auto empty = card("contentCard");
                auto layout = new QVBoxLayout(empty);
                layout->setContentsMargins(18, 18, 18, 18);
                auto label = new QLabel(tr("A little space for your thoughts\nWrite a note above or choose another filter."));
                label->setObjectName("emptyState");
                label->setAlignment(Qt::AlignCenter);
                label->setMinimumHeight(150);
                layout->addWidget(label);
                rowLayout->addWidget(empty);
            }

            if (!append && restoreScroll >= 0) {
                QTimer::singleShot(0, scroll, [this, current, anchorUid, anchorViewportY, anchorFound, restoreScroll] {
                    if (current != generation)
                        return;
                    rowLayout->activate();
                    int target = restoreScroll;
                    if (anchorFound) {
                        for (int i = 0; i < rowLayout->count(); ++i) {
                            auto widget = rowLayout->itemAt(i)->widget();
                            if (widget && widget->property("memoUid").toString() == anchorUid) {
                                const int cardY = widget->mapTo(rows, QPoint(0, 0)).y();
                                target = cardY - anchorViewportY;
                                break;
                            }
                        }
                    }
                    auto bar = scroll->verticalScrollBar();
                    bar->setValue(qBound(0, target, bar->maximum()));
                });
            }
        });
}
void MainWindow::addNote(const QJsonObject &memo) {
    auto f = card("noteCard");
    auto l = new QVBoxLayout(f);
    l->setContentsMargins(14, 12, 14, 13);
    l->setSpacing(9);
    auto top = new QHBoxLayout;
    auto stamp = new QLabel(QDateTime::fromString(memo["createdAt"].toString(), Qt::ISODate)
                                .toLocalTime()
                                .toString("dd MMM yyyy · HH:mm"));
    stamp->setObjectName("timestamp");
    top->addWidget(stamp);
    top->addStretch();
    auto favorite = iconButton(Glyph::Star, tr("Favorite"));
    favorite->setCheckable(true);
    favorite->setChecked(memo["favorite"].toBool());
    auto edit = button(tr("Edit"));
    edit->setObjectName("textAction");
    edit->setIcon(lineIcon(Glyph::Edit));
    edit->setIconSize({18, 18});
    auto menu = iconButton(Glyph::More, tr("More actions"));
    top->addWidget(favorite);
    top->addWidget(edit);
    top->addWidget(menu);
    l->addLayout(top);
    const auto uid = memo["uid"].toString();
    f->setProperty("memoUid", uid);
    connect(favorite, &QToolButton::toggled, this, [this, uid, favorite](bool checked) {
        favorite->setEnabled(false);
        backend.call(favorite, "memos.favorite", {{"uid", uid}, {"favorite", checked}},
                     [this, favorite, checked](const QJsonValue &, const QString &code, const QString &message) {
                         favorite->setEnabled(true);
                         if (!code.isEmpty()) {
                             QSignalBlocker blocker(favorite);
                             favorite->setChecked(!checked);
                             showError(message);
                             return;
                         }
                         metadata();
                     });
    });
    connect(edit, &QPushButton::clicked, this, [this, uid] { editNote(uid); });
    connect(menu, &QToolButton::clicked, this, [this, uid, menu] {
        QMenu m;
        auto share = m.addAction(lineIcon(Glyph::Share), tr("Share"));
        auto remove = m.addAction(lineIcon(Glyph::Trash), tr("Delete"));
        auto chosen = m.exec(menu->mapToGlobal(QPoint(0, menu->height())));
        if (chosen == share)
            shareNote(uid);
        if (chosen == remove &&
            QMessageBox::question(this, tr("Delete note?"), tr("Delete this note and its attachments?")) ==
                QMessageBox::Yes)
            rpc("memos.delete", {{"uid", uid}}, [this](const QJsonValue &) {
                refresh();
                metadata();
            });
    });

    auto text = new ContainedTextBrowser;
    text->setObjectName("notePreview");
    text->setDocument(new LocalTextDocument(text));
    text->document()->setDocumentMargin(0);
    text->setOpenLinks(false);
    text->setOpenExternalLinks(false);
    text->setFrameShape(QFrame::NoFrame);
    text->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    text->setVerticalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
    text->document()->setDefaultStyleSheet(
        "p { margin-top: 0; margin-bottom: 6px; } "
        "ul, ol { margin-top: 4px; margin-bottom: 6px; } "
        "pre { margin-top: 6px; margin-bottom: 6px; }");
    loadNoteDocument(text->document(), memo["content"].toString());
    text->setMinimumHeight(36);
    text->setMaximumHeight(220);
    text->setSizePolicy(QSizePolicy::Expanding, QSizePolicy::Preferred);
    auto fitPreview = [text](QSizeF size) {
        const int height = qBound(38, int(std::ceil(size.height())) + 8, 220);
        text->setFixedHeight(height);
    };
    connect(text->document()->documentLayout(), &QAbstractTextDocumentLayout::documentSizeChanged, text, fitPreview);
    QTimer::singleShot(0, text, [text, fitPreview] { fitPreview(text->document()->size()); });
    l->addWidget(text);

    const auto files = memo["attachments"].toArray();
    bool images = false;
    for (const auto &a : files)
        images |= a.toObject()["mimeType"].toString().startsWith("image/");
    if (images) {
        auto gallery = new ImageGallery(files);
        l->addWidget(gallery);
        connect(gallery, &ImageGallery::opened, this,
                [](const QString &p) { QDesktopServices::openUrl(QUrl::fromLocalFile(p)); });
        connect(gallery, &ImageGallery::reordered, this, [this, uid, gallery](const QJsonArray &order) {
            gallery->setEnabled(false);
            QPointer<ImageGallery> guard(gallery);
            backend.call(this, "attachments.reorder", {{"uid", uid}, {"order", order}},
                         [this, guard, uid](const QJsonValue &, const QString &code, const QString &message) {
                             if (guard)
                                 guard->setEnabled(true);
                             if (!code.isEmpty())
                                 showError(message);
                             else
                                 refresh(false, uid);
                         });
        });
    }
    auto fileRow = new QWidget;
    auto fileLayout = new QHBoxLayout(fileRow);
    fileLayout->setContentsMargins(0, 0, 0, 0);
    fileLayout->setSpacing(6);
    int fileCount = 0;
    for (const auto &value : files) {
        auto a = value.toObject();
        if (a["mimeType"].toString().startsWith("image/"))
            continue;
        auto b = button(a["filename"].toString());
        b->setObjectName("fileChip");
        b->setIcon(lineIcon(Glyph::Attachment));
        b->setToolTip(a["path"].toString());
        fileLayout->addWidget(b);
        ++fileCount;
        connect(b, &QPushButton::clicked, this,
                [a] { QDesktopServices::openUrl(QUrl::fromLocalFile(a["path"].toString())); });
    }
    if (fileCount) {
        fileLayout->addStretch();
        l->addWidget(fileRow);
    } else {
        delete fileRow;
    }

    const auto memoTags = memo["tags"].toArray();
    if (!memoTags.isEmpty()) {
        auto tagsWidget = new QWidget;
        auto grid = new QGridLayout(tagsWidget);
        grid->setContentsMargins(0, 0, 0, 0);
        grid->setHorizontalSpacing(6);
        grid->setVerticalSpacing(5);
        int i = 0;
        for (const auto &t : memoTags) {
            const auto tagName = t.toString();
            auto chip = button("#" + tagName);
            chip->setObjectName("tagChip");
            grid->addWidget(chip, i / 4, i % 4, Qt::AlignLeft);
            ++i;
            connect(chip, &QPushButton::clicked, this, [this, tagName] {
                tag = tagName;
                page = "home";
                refresh();
            });
        }
        grid->setColumnStretch(4, 1);
        l->addWidget(tagsWidget);
    }
    rowLayout->addWidget(f);
}
void MainWindow::addFiles(const QString &uid, const QStringList &paths, std::function<void()> done) {
    rpc("attachments.add", {{"uid", uid}, {"paths", jsonPaths(paths)}}, [this, done](const QJsonValue &v) {
        auto failures = fileFailures(v);
        if (!failures.isEmpty())
            showError(tr("The note was saved, but some attachments failed:\n") + failures.join('\n'));
        if (done)
            done();
    });
}

void MainWindow::editNote(const QString &uid) {
    rpc("memos.get", {{"uid", uid}}, [this, uid](const QJsonValue &v) {
        auto memo = v.toObject();
        auto dialog = new MemoDialog(this);
        dialog->setAttribute(Qt::WA_DeleteOnClose);
        dialog->setWindowTitle(tr("Edit Note"));
        dialog->resize(qMin(940, width() - 60), qMin(760, height() - 60));
        dialog->setMinimumSize(560, 420);

        auto layout = new QVBoxLayout(dialog);
        layout->setContentsMargins(14, 14, 14, 12);
        layout->setSpacing(10);

        auto editor = new NoteEditor;
        dialog->editor = editor;
        editor->setContent(memo["content"].toString());
        layout->addWidget(editor, 1);

        auto status = new QLabel;
        status->setObjectName("muted");
        status->setWordWrap(true);
        status->hide();
        layout->addWidget(status);

        auto buttons = new QHBoxLayout;
        auto cancel = button(tr("Cancel")), saveNote = button(tr("Save"));
        saveNote->setObjectName("primary");
        buttons->addStretch();
        buttons->addWidget(cancel);
        buttons->addWidget(saveNote);
        layout->addLayout(buttons);

        auto current = std::make_shared<QJsonObject>(memo);
        auto setStatus = [status](const QString &message) {
            status->setText(message);
            status->setVisible(!message.trimmed().isEmpty());
        };
        auto setBusy = [dialog, editor, cancel, saveNote](bool busy) {
            dialog->busy = busy;
            for (auto widget : QList<QWidget *>{editor, cancel, saveNote})
                widget->setEnabled(!busy);
        };

        connect(cancel, &QPushButton::clicked, dialog, &QDialog::reject);
        connect(editor, &NoteEditor::attachRequested, dialog,
                [this, dialog, current, uid, setBusy, setStatus] {
                    const auto paths = QFileDialog::getOpenFileNames(dialog, tr("Attach files"));
                    if (paths.isEmpty())
                        return;
                    setBusy(true);
                    setStatus(tr("Attaching files…"));
                    backend.call(dialog, "attachments.add", {{"uid", uid}, {"paths", jsonPaths(paths)}},
                                 [this, current, uid, paths, setBusy, setStatus](const QJsonValue &result,
                                                                                const QString &code,
                                                                                const QString &message) {
                                     setBusy(false);
                                     if (!code.isEmpty()) {
                                         setStatus(message);
                                         return;
                                     }
                                     const auto object = result.toObject();
                                     const auto updated = object["memo"].toObject();
                                     if (!updated.isEmpty())
                                         *current = updated;
                                     const auto failures = fileFailures(result);
                                     if (failures.isEmpty())
                                         setStatus(tr("Attached %1 file(s).").arg(paths.size()));
                                     else
                                         setStatus(tr("Some files could not be attached:\n%1").arg(failures.join('\n')));
                                     refresh(false, uid);
                                 });
                });

        connect(saveNote, &QPushButton::clicked, dialog,
                [this, dialog, editor, current, uid, setBusy, setStatus] {
                    setBusy(true);
                    setStatus(tr("Saving…"));
                    backend.call(dialog, "memos.update",
                                 {{"uid", uid}, {"revision", (*current)["revision"]}, {"content", editor->content()}},
                                 [this, dialog, uid, setBusy, setStatus](const QJsonValue &, const QString &code,
                                                                        const QString &message) {
                                     setBusy(false);
                                     if (!code.isEmpty()) {
                                         setStatus(code == "conflict"
                                                       ? tr("This note changed elsewhere. Your draft is kept. "
                                                            "Copy it before closing and reopen the note to compare changes.")
                                                       : message);
                                         return;
                                     }
                                     dialog->accept();
                                     refresh(false, uid);
                                     metadata();
                                 });
                });
        dialog->open();
    });
}
void MainWindow::shareNote(const QString &uid) {
    auto dialog = new QDialog(this);
    dialog->setAttribute(Qt::WA_DeleteOnClose);
    dialog->setWindowTitle(tr("Share note"));
    dialog->resize(560, 360);
    auto l = new QVBoxLayout(dialog);
    auto hint = new QLabel(tr("Links work on this computer while local sharing is enabled in Settings."));
    hint->setWordWrap(true);
    l->addWidget(hint);
    auto list = new QListWidget;
    l->addWidget(list);
    auto expiry = new QComboBox;
    expiry->addItems({tr("No expiry"), tr("1 day"), tr("7 days"), tr("30 days")});
    l->addWidget(expiry);
    auto url = new QLineEdit;
    url->setReadOnly(true);
    url->setPlaceholderText(tr("New link appears here once — copy it now"));
    l->addWidget(url);
    auto create = button(tr("Create link")), revoke = button(tr("Revoke selected"));
    l->addWidget(create);
    l->addWidget(revoke);
    auto load = [this, dialog, list, uid] {
        backend.call(dialog, "shares.list", {{"uid", uid}},
                     [this, list](const QJsonValue &v, const QString &code, const QString &message) {
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         list->clear();
                         for (const auto &item : v.toArray()) {
                             auto s = item.toObject();
                             auto row = new QListWidgetItem(
                                 s["uid"].toString() + " · " +
                                     (s["expiresAt"].isNull() ? tr("No expiry") : s["expiresAt"].toString()),
                                 list);
                             row->setData(Qt::UserRole, s["uid"].toString());
                         }
                     });
    };
    connect(create, &QPushButton::clicked, dialog, [this, dialog, uid, expiry, url, load] {
        const int days[] = {0, 1, 7, 30};
        backend.call(dialog, "shares.create", {{"uid", uid}, {"days", days[expiry->currentIndex()]}},
                     [this, url, load](const QJsonValue &v, const QString &code, const QString &message) {
                         if (!code.isEmpty()) {
                             showError(message);
                             return;
                         }
                         url->setText(v.toObject()["url"].toString());
                         url->selectAll();
                         load();
                         metadata();
                     });
    });
    connect(revoke, &QPushButton::clicked, dialog, [this, dialog, list, load] {
        if (!list->currentItem())
            return;
        backend.call(dialog, "shares.revoke", {{"uid", list->currentItem()->data(Qt::UserRole).toString()}},
                     [this, load](const QJsonValue &, const QString &code, const QString &message) {
                         if (!code.isEmpty())
                             showError(message);
                         else {
                             load();
                             metadata();
                         }
                     });
    });
    load();
    dialog->open();
}
void MainWindow::settingsPage() {
    auto appearanceCard = card("settingsCard");
    auto appearanceLayout = new QVBoxLayout(appearanceCard);
    appearanceLayout->setContentsMargins(16, 14, 16, 16);
    appearanceLayout->setSpacing(10);
    auto appearanceTitle = new QLabel(tr("Appearance"));
    appearanceTitle->setObjectName("sectionTitle");
    auto appearanceHint = new QLabel(tr("Choose a theme and typography that stay comfortable for long note sessions."));
    appearanceHint->setObjectName("muted");
    appearanceHint->setWordWrap(true);
    appearanceLayout->addWidget(appearanceTitle);
    appearanceLayout->addWidget(appearanceHint);
    auto form = new QFormLayout;
    form->setHorizontalSpacing(18);
    form->setVerticalSpacing(10);
    form->setFieldGrowthPolicy(QFormLayout::FieldsStayAtSizeHint);
    form->setFormAlignment(Qt::AlignLeft | Qt::AlignTop);
    form->setLabelAlignment(Qt::AlignLeft | Qt::AlignVCenter);

    auto appearance = new ArrowComboBox;
    appearance->setObjectName("appearanceCombo");
    appearance->addItems({"System", "Light", "Dark"});
    appearance->setCurrentText(settings.value("appearance", "Light").toString());
    appearance->setMinimumWidth(190);
    appearance->setMaximumWidth(230);
    appearance->setSizeAdjustPolicy(QComboBox::AdjustToContents);
    appearance->setToolTip(tr("Follow the system theme or choose a fixed appearance."));
    form->addRow(tr("Theme"), appearance);
    connect(appearance, &QComboBox::currentTextChanged, this, [this](const QString &v) {
        settings.setValue("appearance", v);
        applyAppearance();
    });

    auto font = new ArrowFontComboBox;
    font->setObjectName("fontCombo");
    font->setEditable(true);
    font->setInsertPolicy(QComboBox::NoInsert);
    font->setCurrentFont(QFont(settings.value("fontFamily", qApp->font().family()).toString()));
    font->setMinimumWidth(270);
    font->setMaximumWidth(340);
    font->setMaxVisibleItems(18);
    font->setSizeAdjustPolicy(QComboBox::AdjustToMinimumContentsLengthWithIcon);
    font->setMinimumContentsLength(22);
    if (font->completer()) {
        font->completer()->setCaseSensitivity(Qt::CaseInsensitive);
        font->completer()->setCompletionMode(QCompleter::PopupCompletion);
        font->completer()->setFilterMode(Qt::MatchContains);
    }
    if (font->view()) {
        font->view()->setMinimumWidth(font->minimumWidth());
        font->view()->setMaximumWidth(font->maximumWidth());
        font->view()->setHorizontalScrollBarPolicy(Qt::ScrollBarAlwaysOff);
        font->view()->setTextElideMode(Qt::ElideRight);
    }
    font->setToolTip(tr("Type part of a font name or open the list to browse installed fonts."));
    form->addRow(tr("Font"), font);
    connect(font, &QFontComboBox::currentFontChanged, this, [this](const QFont &f) {
        if (f.family().trimmed().isEmpty())
            return;
        settings.setValue("fontFamily", f.family());
        applyAppearance();
    });

    auto size = new ArrowSpinBox;
    size->setObjectName("fontSizeSpin");
    size->setRange(9, 24);
    size->setValue(settings.value("fontSize", 10).toInt());
    size->setSuffix(tr(" pt"));
    size->setMinimumWidth(105);
    size->setMaximumWidth(120);
    size->setAlignment(Qt::AlignLeft);
    form->addRow(tr("Text size"), size);
    connect(size, &QSpinBox::valueChanged, this, [this](int n) {
        settings.setValue("fontSize", n);
        applyAppearance();
    });
    appearanceLayout->addLayout(form);
    rowLayout->addWidget(appearanceCard);

    auto dataCard = card("settingsCard");
    auto dataLayout = new QVBoxLayout(dataCard);
    dataLayout->setContentsMargins(16, 14, 16, 16);
    dataLayout->setSpacing(10);
    auto dataTitle = new QLabel(tr("Data & backup"));
    dataTitle->setObjectName("sectionTitle");
    auto dataHint = new QLabel(tr("Your database and attachments stay local. Export a ZIP before moving devices or making major changes."));
    dataHint->setObjectName("muted");
    dataHint->setWordWrap(true);
    dataLayout->addWidget(dataTitle);
    dataLayout->addWidget(dataHint);
    auto dataActions = new QHBoxLayout;
    auto folder = button(tr("Open data folder"));
    folder->setObjectName("outlineButton");
    folder->setIcon(lineIcon(Glyph::Folder));
    auto exportFile = button(tr("Export ZIP backup"));
    exportFile->setObjectName("outlineButton");
    auto importFile = button(tr("Import ZIP backup"));
    importFile->setObjectName("outlineButton");
    dataActions->addWidget(folder);
    dataActions->addWidget(exportFile);
    dataActions->addWidget(importFile);
    dataActions->addStretch();
    dataLayout->addLayout(dataActions);
    connect(folder, &QPushButton::clicked, this, [this] { QDesktopServices::openUrl(QUrl::fromLocalFile(dataPath)); });
    connect(exportFile, &QPushButton::clicked, this, [this] {
        auto path = QFileDialog::getSaveFileName(
            this, tr("Export to a new file"),
            "NoteHub-" + QDate::currentDate().toString(Qt::ISODate) + ".zip", tr("Backup (*.zip)"));
        if (path.isEmpty())
            return;
        rpc("backup.export", {{"path", path}}, [this](const QJsonValue &) {
            QMessageBox::information(this, tr("Backup"), tr("Backup exported."));
        });
    });
    connect(importFile, &QPushButton::clicked, this, [this] {
        auto path = QFileDialog::getOpenFileName(this, tr("Import backup"), {}, tr("Backup (*.zip)"));
        if (path.isEmpty())
            return;
        bool ok;
        auto policy = QInputDialog::getItem(this, tr("Conflicting notes"), tr("Import policy"),
                                            {"skip", "replace", "duplicate"}, 0, false, &ok);
        if (!ok)
            return;
        rpc("backup.import", {{"path", path}, {"policy", policy}}, [this](const QJsonValue &v) {
            QMessageBox::information(
                this, tr("Import result"),
                QString::fromUtf8(QJsonDocument(v.toObject()).toJson(QJsonDocument::Indented)));
            metadata();
            refresh();
        });
    });
    rowLayout->addWidget(dataCard);

    auto sharingCard = card("settingsCard");
    auto sharingLayout = new QVBoxLayout(sharingCard);
    sharingLayout->setContentsMargins(16, 14, 16, 16);
    sharingLayout->setSpacing(10);
    auto sharingTitle = new QLabel(tr("Local sharing"));
    sharingTitle->setObjectName("sectionTitle");
    auto sharingHint = new QLabel(tr("Share links are served only on this computer and the server starts only when you enable it."));
    sharingHint->setObjectName("muted");
    sharingHint->setWordWrap(true);
    sharingLayout->addWidget(sharingTitle);
    sharingLayout->addWidget(sharingHint);
    auto sharingRow = new QHBoxLayout;
    auto sharing = new QCheckBox(tr("Enable local sharing"));
    auto portLabel = new QLabel(tr("Port"));
    portLabel->setObjectName("muted");
    auto port = new QSpinBox;
    port->setRange(0, 65535);
    port->setValue(settings.value("sharingPort", 8787).toInt());
    port->setSpecialValueText(tr("Auto"));
    port->setMaximumWidth(140);
    sharingRow->addWidget(sharing);
    sharingRow->addStretch();
    sharingRow->addWidget(portLabel);
    sharingRow->addWidget(port);
    sharingLayout->addLayout(sharingRow);
    QPointer<QCheckBox> guard(sharing);
    backend.call(sharing, "metadata", {}, [guard](const QJsonValue &v, const QString &code, const QString &) {
        if (guard && code.isEmpty()) {
            QSignalBlocker b(guard);
            guard->setChecked(!v.toObject()["sharingUrl"].toString().isEmpty());
        }
    });
    connect(sharing, &QCheckBox::toggled, this, [this, sharing, port](bool enabled) {
        sharing->setEnabled(false);
        settings.setValue("sharingPort", port->value());
        QPointer<QCheckBox> guard(sharing);
        backend.call(sharing, "sharing.set", {{"enabled", enabled}, {"port", port->value()}},
                     [this, guard, enabled](const QJsonValue &, const QString &code, const QString &message) {
                         if (!guard)
                             return;
                         guard->setEnabled(true);
                         if (!code.isEmpty()) {
                             QSignalBlocker b(guard);
                             guard->setChecked(!enabled);
                             showError(message);
                         }
                         metadata();
                     });
    });
    auto version = new QLabel(tr("NoteHub %1  ·  Qt desktop + Go core").arg(NOTEHUB_VERSION));
    version->setObjectName("muted");
    sharingLayout->addWidget(version);
    rowLayout->addWidget(sharingCard);
}
void MainWindow::applyAppearance() {
    const auto mode = settings.value("appearance", "Light").toString();
    const bool dark = mode == "Dark" ||
                      (mode == "System" && QGuiApplication::styleHints()->colorScheme() == Qt::ColorScheme::Dark);

    const QString requestedFamily = settings.value("fontFamily", qApp->font().family()).toString();
    const int pointSize = settings.value("fontSize", 10).toInt();
    const auto installedFamilies = QFontDatabase::families();
    const QString family = installedFamilies.contains(requestedFamily, Qt::CaseInsensitive)
                               ? requestedFamily
                               : qApp->font().family();
    qApp->setFont(QFont(family, pointSize));

    // One semantic palette drives widgets, custom icons, calendars and custom painting.
    // This avoids the old half-light/half-dark state where static icon pixmaps and
    // QCalendarWidget kept colors from the theme that was active at construction.
    const QString bg = dark ? "#0d1521" : "#f5f7fb";
    const QString surface = dark ? "#151f2d" : "#ffffff";
    const QString surfaceAlt = dark ? "#1a2738" : "#f8fafc";
    const QString rail = dark ? "#172435" : "#eef3f9";
    const QString border = dark ? "#2d3d52" : "#dfe6ef";
    const QString text = dark ? "#e8eef7" : "#172033";
    const QString muted = dark ? "#95a6bc" : "#6b778c";
    const QString disabled = dark ? "#64758b" : "#9aa6b5";
    const QString blue = dark ? "#3b97ff" : "#087bff";
    const QString blueSoft = dark ? "#183e69" : "#e7f2ff";
    const QString hover = dark ? "#22344a" : "#f0f5fb";

    QPalette palette;
    palette.setColor(QPalette::Window, QColor(bg));
    palette.setColor(QPalette::WindowText, QColor(text));
    palette.setColor(QPalette::Base, QColor(surface));
    palette.setColor(QPalette::AlternateBase, QColor(surfaceAlt));
    palette.setColor(QPalette::ToolTipBase, QColor(surface));
    palette.setColor(QPalette::ToolTipText, QColor(text));
    palette.setColor(QPalette::Text, QColor(text));
    palette.setColor(QPalette::Button, QColor(surface));
    palette.setColor(QPalette::ButtonText, QColor(text));
    palette.setColor(QPalette::BrightText, QColor("#ffffff"));
    palette.setColor(QPalette::Light, QColor(surfaceAlt));
    palette.setColor(QPalette::Midlight, QColor(border));
    palette.setColor(QPalette::Mid, QColor(border));
    palette.setColor(QPalette::Dark, QColor(border));
    palette.setColor(QPalette::Shadow, QColor(dark ? "#05080d" : "#c6ced9"));
    palette.setColor(QPalette::PlaceholderText, QColor(muted));
    palette.setColor(QPalette::Highlight, QColor(blue));
    palette.setColor(QPalette::HighlightedText, QColor("#ffffff"));
    palette.setColor(QPalette::Link, QColor(blue));
    palette.setColor(QPalette::LinkVisited, QColor(blue));
    palette.setColor(QPalette::Disabled, QPalette::Text, QColor(disabled));
    palette.setColor(QPalette::Disabled, QPalette::WindowText, QColor(disabled));
    palette.setColor(QPalette::Disabled, QPalette::ButtonText, QColor(disabled));
    palette.setColor(QPalette::Disabled, QPalette::PlaceholderText, QColor(disabled));
    qApp->setPalette(palette);
    QPixmapCache::clear();

    qApp->setStyleSheet(QString(R"QSS(
        * { outline: none; }
        QWidget { color:%1; }
        QWidget#appRoot { background:%2; }
        QSplitter#navigationSplitter { background:transparent; }
        QSplitter#navigationSplitter::handle:horizontal {
            background:transparent; width:7px; margin:10px 1px; border-radius:2px;
        }
        QSplitter#navigationSplitter::handle:horizontal:hover { background:%6; }
        QSplitter#navigationSplitter::handle:horizontal:pressed { background:%12; }
        QWidget#contentColumns { background:transparent; }
        QFrame#workspace { background:%3; border:1px solid %4; border-radius:14px; }
        QFrame#sideRail { background:%5; border:1px solid %4; border-radius:14px; }
        QWidget#rightColumn { background:transparent; }

        QFrame#card, QFrame#contentCard, QFrame#composerCard, QFrame#noteCard,
        QFrame#sideCard, QFrame#settingsCard, QFrame#attachmentRow {
            background:%3; border:1px solid %4; border-radius:12px;
        }
        QFrame#noteCard:hover, QFrame#attachmentRow:hover { border-color:%6; }
        QFrame#quickRow { background:transparent; border:0; border-radius:8px; }
        QFrame#quickRow:hover { background:%7; }
        QFrame#railSeparator {
            color:%4; background:%4; max-height:1px; border:0; margin:5px 4px;
        }

        QLabel#brand { font-size:26px; font-weight:700; letter-spacing:-0.4px; }
        QLabel#heading { font-size:18px; font-weight:700; }
        QLabel#sideTitle, QLabel#sectionTitle { font-size:14px; font-weight:700; }
        QLabel#muted, QLabel#timestamp, QLabel#stagedFiles { color:%8; }
        QLabel#timestamp { font-size:12px; }
        QLabel#stagedFiles { font-size:12px; padding-left:4px; }
        QLabel#emptyState { color:%8; font-size:13px; }
        QLabel#countBadge {
            background:%9; color:%6; border-radius:10px; min-width:24px; max-width:36px;
            padding:2px 5px; font-weight:700;
        }

        QLineEdit#globalSearch {
            background:%3; border:1px solid %4; border-radius:11px; padding:0 12px;
        }
        QLineEdit#globalSearch:focus { border:1px solid %6; }
        QTextEdit#composer {
            background:%3; border:1px solid %4; border-radius:9px; padding:8px;
        }
        QTextEdit#composer:focus { border:1px solid %6; }
        QTextBrowser#notePreview { background:transparent; border:0; padding:0; }

        QPushButton, QToolButton {
            border:0; border-radius:8px; padding:7px 10px; background:transparent;
        }
        QPushButton:hover, QToolButton:hover { background:%7; }
        QPushButton:pressed, QToolButton:pressed { background:%9; }
        QPushButton:disabled, QToolButton:disabled { color:%11; }
        QPushButton#primary {
            background:%6; color:white; font-weight:700; padding:8px 15px;
        }
        QPushButton#primary:hover { background:%12; }
        QPushButton#outlineButton {
            background:%3; border:1px solid %4; padding:7px 11px;
        }
        QPushButton#outlineButton:hover { border-color:%6; background:%9; }
        QPushButton#subtleButton { color:%8; }
        QPushButton#composerAction { font-weight:600; padding:6px 8px; }
        QPushButton#textAction { font-weight:600; padding:5px 8px; }
        QPushButton#tagChip {
            background:%9; color:%6; border-radius:12px; padding:4px 9px; font-weight:600;
        }
        QPushButton#fileChip { background:%7; border:1px solid %4; padding:5px 9px; }
        QPushButton#fileLink { text-align:left; padding:0; font-weight:600; }
        QPushButton#fileLink:hover { color:%6; background:transparent; }
        QPushButton#tagCard {
            text-align:left; background:%10; border:1px solid %4; padding:10px 12px; font-weight:600;
        }
        QPushButton#tagCard:hover { border-color:%6; background:%9; }
        QPushButton#headerIconButton { padding:7px; }
        QToolButton#iconButton, QToolButton#dangerIconButton { padding:6px; }
        QToolButton#decorativeIcon { padding:4px; background:transparent; }
        QToolButton#decorativeIcon:hover { background:transparent; }
        QToolButton#dangerIconButton:hover { background:rgba(214,63,63,0.12); }
        QToolButton#navButton {
            padding:9px 10px; color:%8;
        }
        QToolButton#navButton[compact="true"] { text-align:center; padding:9px; }
        QToolButton#navButton[compact="false"] { text-align:left; padding:9px 10px; }
        QToolButton#navButton:hover { color:%1; }
        QToolButton#navButton:checked {
            background:%9; color:%6; font-weight:700;
        }
        QToolButton#quickButton { text-align:left; padding:7px 4px; }

        QLineEdit, QPlainTextEdit, QTextEdit, QComboBox, QSpinBox, QFontComboBox {
            background:%3; color:%1; border:1px solid %4; border-radius:8px;
            padding:6px 8px; selection-background-color:%6; selection-color:white;
        }
        QLineEdit:focus, QPlainTextEdit:focus, QTextEdit:focus, QComboBox:focus,
        QSpinBox:focus, QFontComboBox:focus { border-color:%6; }
        QLineEdit:disabled, QPlainTextEdit:disabled, QTextEdit:disabled, QComboBox:disabled,
        QSpinBox:disabled, QFontComboBox:disabled { color:%11; background:%10; }
        QComboBox::drop-down, QFontComboBox::drop-down {
            border:0; width:30px; subcontrol-origin:padding; subcontrol-position:top right;
        }
        QComboBox::down-arrow, QFontComboBox::down-arrow { image:none; width:0; height:0; }
        QSpinBox#fontSizeSpin { padding-right:32px; }
        QToolButton#spinIncreaseButton, QToolButton#spinDecreaseButton {
            border:0; padding:0; margin:0; border-radius:4px; background:transparent;
        }
        QToolButton#spinIncreaseButton:hover, QToolButton#spinDecreaseButton:hover { background:%7; }
        QToolButton#spinIncreaseButton:pressed, QToolButton#spinDecreaseButton:pressed { background:%9; }
        QToolButton#spinIncreaseButton:disabled, QToolButton#spinDecreaseButton:disabled { background:transparent; }
        QComboBox QAbstractItemView, QFontComboBox QAbstractItemView {
            background:%3; color:%1; border:1px solid %4; border-radius:8px;
            selection-background-color:%9; selection-color:%6; padding:4px;
        }
        QComboBox QAbstractItemView::item, QFontComboBox QAbstractItemView::item {
            min-height:25px; padding:3px 6px;
        }

        QCheckBox { spacing:8px; }
        QCheckBox::indicator { width:17px; height:17px; }

        QScrollArea, QWidget#rows, QScrollArea#navScroll, QScrollArea#contentScroll {
            border:0; background:transparent;
        }
        QScrollBar:vertical { background:transparent; width:8px; margin:2px; }
        QScrollBar::handle:vertical { background:%4; min-height:28px; border-radius:4px; }
        QScrollBar::handle:vertical:hover { background:%8; }
        QScrollBar::add-line:vertical, QScrollBar::sub-line:vertical { height:0; }

        /* QCalendarWidget owns an internal QTableView. Never make that view
           transparent: on Windows/Fusion its viewport can otherwise expose a
           black backing store in a light theme. */
        QCalendarWidget {
            background:%3; color:%1; border:0;
        }
        QCalendarWidget QWidget#qt_calendar_navigationbar {
            background:%3; color:%1; border:0;
        }
        QCalendarWidget QToolButton {
            color:%1; font-weight:700; padding:5px; background:transparent;
        }
        QCalendarWidget QToolButton:hover { background:%7; }
        QCalendarWidget QMenu { background:%3; color:%1; border:1px solid %4; }
        QCalendarWidget QSpinBox {
            background:%3; color:%1; border:0; selection-background-color:%9;
        }
        QCalendarWidget QTableView, QCalendarWidget QAbstractItemView {
            background:%3; alternate-background-color:%3; color:%1; border:0;
            outline:0; selection-background-color:%9; selection-color:%6;
        }
        QCalendarWidget QTableView::item, QCalendarWidget QAbstractItemView::item {
            background:%3; border-radius:6px; padding:3px;
        }
        QCalendarWidget QTableView::item:selected, QCalendarWidget QAbstractItemView::item:selected {
            background:%9; color:%6;
        }
        QCalendarWidget QTableView::item:hover, QCalendarWidget QAbstractItemView::item:hover {
            background:%7;
        }

        QMenu { background:%3; color:%1; border:1px solid %4; padding:5px; }
        QMenu::item { padding:7px 22px 7px 10px; border-radius:6px; }
        QMenu::item:selected { background:%9; color:%6; }
        QMenu::separator { height:1px; background:%4; margin:4px 7px; }
        QToolTip { background:%3; color:%1; border:1px solid %4; padding:5px; }
        QMessageBox, QDialog { background:%2; color:%1; }
        QListWidget {
            background:%3; color:%1; border:1px solid %4; border-radius:8px; padding:4px;
        }
        QListWidget::item { padding:6px; border-radius:6px; }
        QListWidget::item:selected { background:%9; color:%6; }
        QTabWidget::pane {
            border:1px solid %4; border-radius:8px; background:%3; top:-1px;
        }
        QTabBar::tab { background:transparent; padding:7px 12px; color:%8; }
        QTabBar::tab:selected { color:%6; font-weight:700; border-bottom:2px solid %6; }
        QToolBar#editorToolbar {
            background:%10; border:1px solid %4; border-radius:10px; spacing:2px; padding:4px;
        }
        QToolBar#editorToolbar QToolButton {
            min-width:34px; min-height:34px; padding:4px; border-radius:7px;
        }
        QToolBar#editorToolbar QToolButton:hover { background:%7; }
        QToolBar#editorToolbar QToolButton:pressed,
        QToolBar#editorToolbar QToolButton:checked { background:%9; color:%6; }
        QToolBar#editorToolbar QToolButton#editorSpinUp,
        QToolBar#editorToolbar QToolButton#editorSpinDown {
            min-width:0; min-height:0; padding:0; margin:0; border:0; border-radius:3px; background:transparent;
        }
        QToolBar#editorToolbar QToolButton#editorSpinUp:hover,
        QToolBar#editorToolbar QToolButton#editorSpinDown:hover { background:%7; }
        QSpinBox#editorFontSize { padding-right:28px; }
        QToolBar#editorToolbar::separator {
            background:%4; width:1px; margin:6px 5px;
        }
    )QSS")
        .arg(text)        // %1
        .arg(bg)          // %2
        .arg(surface)     // %3
        .arg(border)      // %4
        .arg(rail)        // %5
        .arg(blue)        // %6
        .arg(hover)       // %7
        .arg(muted)       // %8
        .arg(blueSoft)    // %9
        .arg(surfaceAlt)  // %10
        .arg(disabled)    // %11
        .arg(dark ? "#2c8fff" : "#006fe8")); // %12

    // Existing buttons keep their QIcon objects, but PaletteIconEngine paints
    // from the current application palette on every repaint.
    for (auto button : findChildren<QAbstractButton *>())
        button->update();

    // QCalendarWidget caches several private child palettes and date formats.
    // Re-apply both chrome and date colors immediately when the theme changes.
    for (auto widget : findChildren<QCalendarWidget *>())
        refreshCalendar(widget);

    update();
}

void MainWindow::closeEvent(QCloseEvent *e) {
    if (backend.busy()) {
        QMessageBox::information(
            this, tr("Operation in progress"),
            tr("Please wait for the current save, import or export to finish before closing."));
        e->ignore();
        return;
    }
    if ((!composer->toPlainText().trimmed().isEmpty() || !staged.isEmpty()) &&
        QMessageBox::question(this, tr("Discard draft?"),
                              tr("The Home draft has not been saved. Close anyway?")) != QMessageBox::Yes) {
        e->ignore();
        return;
    }
    for (auto child : findChildren<QDialog *>())
        if (auto dialog = dynamic_cast<MemoDialog *>(child))
            if (dialog->isVisible() && dialog->editor && dialog->editor->modified()) {
                dialog->raise();
                e->ignore();
                return;
            }
    backend.stop();
    e->accept();
}
